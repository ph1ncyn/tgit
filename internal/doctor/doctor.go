// Package doctor сканирует репозиторий на типовые проблемы (см. раздел
// "Git Doctor" в INTERFACE.md) и умеет чинить их одним действием.
package doctor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tgit/internal/gitrepo"
	"tgit/internal/i18n"
)

// Issue — одна найденная проблема с готовым действием для исправления.
type Issue struct {
	Title  string
	Detail string
	fix    func(repo *gitrepo.Repo) error
}

// Fix выполняет исправление проблемы.
func (i Issue) Fix(repo *gitrepo.Repo) error {
	return i.fix(repo)
}

var junkDirNames = map[string]bool{
	"node_modules":  true,
	"__pycache__":   true,
	".venv":         true,
	"venv":          true,
	"dist":          true,
	"build":         true,
	"target":        true,
	".pytest_cache": true,
}

// Scan ищет проблемы в репозитории. Не должен ничего изменять на диске.
func Scan(repo *gitrepo.Repo) ([]Issue, error) {
	var issues []Issue

	junk, err := macJunkFiles(repo)
	if err != nil {
		return nil, err
	}
	if len(junk) > 0 {
		files := junk
		issues = append(issues, Issue{
			Title:  fmt.Sprintf(i18n.T.MacJunkTitleFmt, len(files)),
			Detail: strings.Join(files, ", "),
			fix: func(r *gitrepo.Repo) error {
				return fixMacJunk(r, files)
			},
		})
	}

	dirs, err := unignoredJunkDirs(repo)
	if err != nil {
		return nil, err
	}
	if len(dirs) > 0 {
		found := dirs
		issues = append(issues, Issue{
			Title:  fmt.Sprintf(i18n.T.JunkDirsTitleFmt, len(found)),
			Detail: strings.Join(found, ", "),
			fix: func(r *gitrepo.Repo) error {
				return ensureGitignorePatterns(r.Root, found)
			},
		})
	}

	detached, err := detachedHead(repo)
	if err != nil {
		return nil, err
	}
	if detached {
		issues = append(issues, Issue{
			Title:  i18n.T.DetachedHeadTitle,
			Detail: i18n.T.DetachedHeadDetail,
			fix:    fixDetachedHead,
		})
	}

	if kind, found := mergeOrRebaseInProgress(repo); found {
		if kind == "merge" {
			issues = append(issues, Issue{
				Title:  i18n.T.MergeInProgressTitle,
				Detail: i18n.T.MergeInProgressDetail,
				fix:    func(r *gitrepo.Repo) error { return r.MergeAbort() },
			})
		} else {
			issues = append(issues, Issue{
				Title:  i18n.T.RebaseInProgressTitle,
				Detail: i18n.T.RebaseInProgressDetail,
				fix:    func(r *gitrepo.Repo) error { return r.RebaseAbort() },
			})
		}
	}

	crlf, err := mixedLineEndings(repo)
	if err != nil {
		return nil, err
	}
	if len(crlf) > 0 {
		files := crlf
		issues = append(issues, Issue{
			Title:  fmt.Sprintf(i18n.T.MixedLineEndingsTitleFmt, len(files)),
			Detail: strings.Join(files, ", "),
			fix: func(r *gitrepo.Repo) error {
				return ensureGitattributesPattern(r.Root)
			},
		})
	}

	return issues, nil
}

// detachedHead сообщает, что репозиторий сейчас не на ветке.
func detachedHead(repo *gitrepo.Repo) (bool, error) {
	branch, err := repo.CurrentBranch()
	if err != nil {
		return false, err
	}
	return branch == "HEAD", nil
}

// fixDetachedHead создаёt ветку на текущем коммите — ничего не теряется, в
// отличие от переключения на существующую ветку, которое увело бы HEAD.
func fixDetachedHead(repo *gitrepo.Repo) error {
	name := "tgit-recovered-" + time.Now().Format("20060102-150405")
	return repo.CreateBranch(name)
}

// gitDir возвращает путь к каталогу .git репозитория. Если .git — не
// каталог (worktree/submodule, где это файл-указатель), возвращает false —
// такие репозитории явно не поддерживаются этим правилом.
func gitDir(repo *gitrepo.Repo) (string, bool) {
	p := filepath.Join(repo.Root, ".git")
	info, err := os.Stat(p)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return p, true
}

// mergeOrRebaseInProgress определяет незавершённый merge или rebase по
// служебным файлам/каталогам, которые git оставляет в .git до завершения
// или прерывания операции.
func mergeOrRebaseInProgress(repo *gitrepo.Repo) (kind string, found bool) {
	dir, ok := gitDir(repo)
	if !ok {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(dir, "MERGE_HEAD")); err == nil {
		return "merge", true
	}
	if _, err := os.Stat(filepath.Join(dir, "rebase-merge")); err == nil {
		return "rebase", true
	}
	if _, err := os.Stat(filepath.Join(dir, "rebase-apply")); err == nil {
		return "rebase", true
	}
	return "", false
}

// mixedLineEndings ищет отслеживаемые текстовые файлы с CRLF-переносами.
// Бинарные файлы пропускаются, читается не более первых maxBytes каждого
// файла и не более maxFiles файлов за скан — чтобы не тормозить на крупных
// репозиториях. Если .gitattributes уже просит нормализацию, правило молчит:
// иначе оно всплывало бы вечно заново, ведь уже закоммиченные CRLF-байты фикс
// сознательно не переписывает (это была бы правка истории).
func mixedLineEndings(repo *gitrepo.Repo) ([]string, error) {
	if hasLine(filepath.Join(repo.Root, ".gitattributes"), "* text=auto") {
		return nil, nil
	}
	tracked, err := repo.LsFiles()
	if err != nil {
		return nil, err
	}
	const maxFiles = 200
	const maxBytes = 4000
	var found []string
	for i, p := range tracked {
		if i >= maxFiles {
			break
		}
		data, err := readPrefix(filepath.Join(repo.Root, p), maxBytes)
		if err != nil {
			continue
		}
		if bytes.IndexByte(data, 0) >= 0 {
			continue // бинарный файл
		}
		if bytes.Contains(data, []byte("\r\n")) {
			found = append(found, p)
		}
	}
	sort.Strings(found)
	return found, nil
}

func readPrefix(path string, maxBytes int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, maxBytes)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return nil, err
	}
	return buf[:n], nil
}

func hasLine(path, line string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, l := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}

func isMacJunk(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, "._") || base == ".DS_Store"
}

func macJunkFiles(repo *gitrepo.Repo) ([]string, error) {
	tracked, err := repo.LsFiles()
	if err != nil {
		return nil, err
	}
	untracked, err := repo.LsFilesOthers()
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var out []string
	for _, p := range append(tracked, untracked...) {
		if isMacJunk(p) && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out, nil
}

func fixMacJunk(repo *gitrepo.Repo, files []string) error {
	if err := ensureGitignorePatterns(repo.Root, []string{"._*", ".DS_Store"}); err != nil {
		return err
	}
	tracked, err := repo.LsFiles()
	if err != nil {
		return err
	}
	trackedSet := map[string]bool{}
	for _, p := range tracked {
		trackedSet[p] = true
	}

	for _, p := range files {
		if trackedSet[p] {
			_ = repo.Untrack(p)
		}
		_ = os.Remove(filepath.Join(repo.Root, p))
	}
	return nil
}

func unignoredJunkDirs(repo *gitrepo.Repo) ([]string, error) {
	entries, err := repo.LsFilesOthersDirs()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range entries {
		name := strings.TrimSuffix(filepath.Base(strings.TrimSuffix(p, "/")), "/")
		if junkDirNames[name] {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out, nil
}

func ensureGitignorePatterns(root string, patterns []string) error {
	return appendMissingLines(filepath.Join(root, ".gitignore"), patterns)
}

// ensureGitattributesPattern просит git нормализовать окончания строк для
// всех текстовых файлов — не переписывает уже закоммиченные блобы (это было
// бы правкой истории), только меняет поведение для будущих коммитов.
func ensureGitattributesPattern(root string) error {
	return appendMissingLines(filepath.Join(root, ".gitattributes"), []string{"* text=auto"})
}

// appendMissingLines дописывает в path те строки из lines, которых там ещё
// нет, ничего не удаляя и не переупорядочивая — используется и для
// .gitignore, и для .gitattributes.
func appendMissingLines(path string, lines []string) error {
	data, _ := os.ReadFile(path)

	existing := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		existing[strings.TrimSpace(line)] = true
	}

	var toAdd []string
	for _, l := range lines {
		if !existing[l] {
			toAdd = append(toAdd, l)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	var b strings.Builder
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		b.WriteString("\n")
	}
	for _, l := range toAdd {
		b.WriteString(l)
		b.WriteString("\n")
	}
	_, err = f.WriteString(b.String())
	return err
}
