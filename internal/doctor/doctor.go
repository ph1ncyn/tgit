// Package doctor сканирует репозиторий на типовые проблемы (см. раздел
// "Git Doctor" в INTERFACE.md) и умеет чинить их одним действием.
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tgit/internal/gitrepo"
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
			Title:  fmt.Sprintf("Файлы macOS (._*, .DS_Store) в репозитории: %d шт.", len(files)),
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
			Title:  fmt.Sprintf("Служебные каталоги не в .gitignore: %d шт.", len(found)),
			Detail: strings.Join(found, ", "),
			fix: func(r *gitrepo.Repo) error {
				return ensureGitignorePatterns(r.Root, found)
			},
		})
	}

	return issues, nil
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
	path := filepath.Join(root, ".gitignore")
	data, _ := os.ReadFile(path)

	existing := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		existing[strings.TrimSpace(line)] = true
	}

	var toAdd []string
	for _, p := range patterns {
		if !existing[p] {
			toAdd = append(toAdd, p)
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
	for _, p := range toAdd {
		b.WriteString(p)
		b.WriteString("\n")
	}
	_, err = f.WriteString(b.String())
	return err
}
