// Package gitrepo — тонкая кроссплатформенная обёртка над системным git.
// Полагается на установленный git в PATH: это надёжнее и совместимее с
// поведением реального git, чем переизобретать формат объектов через
// отдельную библиотеку.
package gitrepo

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"tgit/internal/i18n"
)

// FileStatus — одна строка `git status --porcelain`.
type FileStatus struct {
	X, Y byte
	Path string
}

// Staged сообщает, застейджен ли файл (индекс отличается от HEAD).
func (f FileStatus) Staged() bool {
	return f.X != ' ' && f.X != '?'
}

// Untracked сообщает, что файл вообще не отслеживается git.
func (f FileStatus) Untracked() bool {
	return f.X == '?' && f.Y == '?'
}

// Commit — одна запись из `git log`.
type Commit struct {
	Hash    string
	Short   string
	Author  string
	Date    string
	Subject string
}

// Repo — открытый git-репозиторий.
type Repo struct {
	Root string
}

// Open находит корень git-репозитория, начиная с dir.
func Open(dir string) (*Repo, error) {
	out, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf(i18n.T.NotGitRepoFmt, err)
	}
	return &Repo{Root: strings.TrimSpace(out)}, nil
}

// Clone клонирует url в dir (текущая директория пользователя — команда
// использует `git clone url .`, поэтому dir должен существовать и быть
// пустым либо отсутствующим в смысле git). При успехе открывает результат
// как обычный репозиторий.
func Clone(url, dir string) (*Repo, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if _, err := runGitCombined(dir, "clone", url, "."); err != nil {
		return nil, fmt.Errorf(i18n.T.CloneFailedFmt, err)
	}
	return Open(dir)
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}

// runGitCombined возвращает stdout+stderr вместе (нужно для команд вроде
// push/pull/fetch, которые печатают сводку в stderr даже при успехе) и не
// прерывает выполнение при ошибке — вызывающий сам решает, что делать.
func runGitCombined(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		if out == "" {
			out = err.Error()
		}
		return "", fmt.Errorf("%s", out)
	}
	return out, nil
}

func splitLines(out string) []string {
	out = strings.TrimRight(out, "\n")
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// CurrentBranch возвращает имя текущей ветки (или "HEAD" при detached HEAD).
func (r *Repo) CurrentBranch() (string, error) {
	out, err := runGit(r.Root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// AheadBehind — насколько текущая ветка опережает/отстаёт от upstream.
// Если upstream не настроен (например, свежесозданная локальная ветка,
// которую ещё ни разу не пушили — как Test-1 в примерах выше), это не
// повод молчать: сравниваем HEAD со всеми remote-tracking ветками, чтобы
// индикатор Push всё равно показал непроталкнутые коммиты. Behind в этом
// случае всегда 0 — тянуть ещё не с чего, upstream не настроен.
func (r *Repo) AheadBehind() (ahead, behind int, err error) {
	out, err := runGit(r.Root, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return r.aheadNoUpstream()
	}
	parts := strings.Fields(out)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("%s", i18n.T.UnexpectedRevListMsg)
	}
	ahead, _ = strconv.Atoi(parts[0])
	behind, _ = strconv.Atoi(parts[1])
	return ahead, behind, nil
}

func (r *Repo) aheadNoUpstream() (ahead, behind int, err error) {
	out, err := runGit(r.Root, "rev-list", "--count", "HEAD", "--not", "--remotes")
	if err != nil {
		return 0, 0, nil // нет ни remotes, ни HEAD (пустой репозиторий) — просто нечего показывать
	}
	ahead, convErr := strconv.Atoi(strings.TrimSpace(out))
	if convErr != nil {
		return 0, 0, nil
	}
	return ahead, 0, nil
}

// Branches возвращает список локальных веток.
func (r *Repo) Branches() ([]string, error) {
	out, err := runGit(r.Root, "branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// Checkout переключает на существующую ветку.
func (r *Repo) Checkout(branch string) error {
	_, err := runGitCombined(r.Root, "checkout", branch)
	return err
}

// CreateBranch создаёт новую ветку от текущего HEAD и сразу переключается на неё.
func (r *Repo) CreateBranch(name string) error {
	_, err := runGitCombined(r.Root, "checkout", "-b", name)
	return err
}

// Status возвращает список изменённых/новых файлов.
func (r *Repo) Status() ([]FileStatus, error) {
	out, err := runGit(r.Root, "status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}
	var files []FileStatus
	for _, line := range splitLines(out) {
		if len(line) < 3 {
			continue
		}
		files = append(files, FileStatus{
			X:    line[0],
			Y:    line[1],
			Path: strings.TrimSpace(line[3:]),
		})
	}
	return files, nil
}

// StageFile добавляет файл в индекс.
func (r *Repo) StageFile(path string) error {
	_, err := runGit(r.Root, "add", "--", path)
	return err
}

// UnstageFile убирает файл из индекса, оставляя изменения в рабочем каталоге.
func (r *Repo) UnstageFile(path string) error {
	_, err := runGit(r.Root, "restore", "--staged", "--", path)
	return err
}

// Commit создаёт коммит из застейдженных изменений.
func (r *Repo) Commit(message string) error {
	_, err := runGitCombined(r.Root, "commit", "-m", message)
	return err
}

// Log возвращает последние limit коммитов текущей ветки.
func (r *Repo) Log(limit int) ([]Commit, error) {
	const sep = "\x1f"
	format := "%H" + sep + "%h" + sep + "%an" + sep + "%ad" + sep + "%s"
	out, err := runGit(r.Root, "log", "-n", strconv.Itoa(limit), "--date=short", "--pretty=format:"+format)
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, err
	}
	var commits []Commit
	for _, line := range splitLines(out) {
		parts := strings.Split(line, sep)
		if len(parts) != 5 {
			continue
		}
		commits = append(commits, Commit{Hash: parts[0], Short: parts[1], Author: parts[2], Date: parts[3], Subject: parts[4]})
	}
	return commits, nil
}

// sanitizeCR убирает голый \r из текста перед показом в терминале. Diff
// файла с CRLF-окончаниями строк несёт \r как часть содержимого КАЖДОЙ
// удалённой строки (git diff отдаёт байты как есть); если пропустить его
// в терминал как есть, тот воспримет "голый" \r посреди строки как
// "вернуть курсор в начало строки" — и следующие байты того же вывода
// затирают уже напечатанное левее на той же физической строке экрана
// (включая соседние панели — Ветки/Файлы делят с Diff одну строку).
// \r\n схлопывается в \n, одиночный оставшийся \r тоже становится переносом.
func sanitizeCR(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// DiffFile возвращает diff одного файла (застейдженный или рабочий, в
// зависимости от staged). Для untracked-файлов diff пуст, поэтому вместо
// него показываем начало содержимого файла как превью.
func (r *Repo) DiffFile(path string, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	out, err := runGit(r.Root, args...)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) != "" {
		return sanitizeCR(out), nil
	}
	return r.untrackedPreview(path)
}

func (r *Repo) untrackedPreview(path string) (string, error) {
	const maxBytes = 4000
	f, err := os.Open(filepath.Join(r.Root, path))
	if err != nil {
		return "", nil // например, файл удалён — просто нет превью
	}
	defer f.Close()

	buf := make([]byte, maxBytes)
	n, _ := f.Read(buf)
	buf = buf[:n]

	for _, b := range buf {
		if b == 0 {
			return i18n.T.BinaryPreviewMsg, nil
		}
	}
	suffix := ""
	if n == maxBytes {
		suffix = i18n.T.TruncatedSuffixMsg
	}
	return sanitizeCR(string(buf)) + suffix, nil
}

// DiffCommit возвращает сообщение коммита и его diff.
func (r *Repo) DiffCommit(hash string) (string, error) {
	out, err := runGit(r.Root, "show", hash)
	if err != nil {
		return "", err
	}
	return sanitizeCR(out), nil
}

// Stash — одна запись `git stash list`.
type Stash struct {
	Ref     string // "stash@{0}"
	Subject string
}

// StashList возвращает список стэшей с их сообщениями, от самого свежего.
func (r *Repo) StashList() ([]Stash, error) {
	const sep = "\x1f"
	out, err := runGit(r.Root, "stash", "list", "--pretty=format:%gd"+sep+"%s")
	if err != nil {
		return nil, err
	}
	var stashes []Stash
	for _, line := range splitLines(out) {
		parts := strings.SplitN(line, sep, 2)
		if len(parts) != 2 {
			continue
		}
		stashes = append(stashes, Stash{Ref: parts[0], Subject: parts[1]})
	}
	return stashes, nil
}

// StashStat — краткая сводка изменённых файлов в стэше (без самого патча).
func (r *Repo) StashStat(ref string) (string, error) {
	return runGit(r.Root, "stash", "show", "--stat", ref)
}

// StashPush сохраняет незакоммиченные изменения в новый стэш.
func (r *Repo) StashPush() (string, error) {
	return runGitCombined(r.Root, "stash", "push")
}

// StashPop возвращает изменения из стэша в рабочий каталог и удаляет запись.
// Пустой ref означает "последний стэш" (поведение `git stash pop` по умолчанию).
func (r *Repo) StashPop(ref string) (string, error) {
	args := []string{"stash", "pop"}
	if ref != "" {
		args = append(args, ref)
	}
	return runGitCombined(r.Root, args...)
}

// StashApply возвращает изменения из стэша в рабочий каталог, но оставляет
// запись в стэше (в отличие от StashPop).
func (r *Repo) StashApply(ref string) (string, error) {
	args := []string{"stash", "apply"}
	if ref != "" {
		args = append(args, ref)
	}
	return runGitCombined(r.Root, args...)
}

// StashDrop безвозвратно удаляет запись стэша, не применяя её.
func (r *Repo) StashDrop(ref string) (string, error) {
	args := []string{"stash", "drop"}
	if ref != "" {
		args = append(args, ref)
	}
	return runGitCombined(r.Root, args...)
}

func (r *Repo) remoteIsGitHubHTTPS() bool {
	out, err := runGit(r.Root, "remote", "get-url", "origin")
	if err != nil {
		return false
	}
	url := strings.TrimSpace(out)
	return strings.HasPrefix(url, "https://github.com/") || strings.HasPrefix(url, "https://www.github.com/")
}

// runGitAuth выполняет git-команду, подставляя GitHub PAT как заголовок
// авторизации для HTTPS-запросов к github.com — так push/pull/fetch работают
// сразу после входа в tgit, без отдельной настройки git credential helper.
// Для остальных remote (SSH, другие хосты) token игнорируется.
func (r *Repo) runGitAuth(token string, args ...string) (string, error) {
	full := args
	if token != "" && r.remoteIsGitHubHTTPS() {
		header := "http.extraHeader=Authorization: Basic " +
			base64.StdEncoding.EncodeToString([]byte("x-access-token:"+token))
		full = append([]string{"-c", header}, args...)
	}
	return runGitCombined(r.Root, full...)
}

func (r *Repo) Push(token string) (string, error) { return r.runGitAuth(token, "push") }

func (r *Repo) Pull(token string) (string, error) {
	out, err := r.runGitAuth(token, "pull", "--prune")
	if err != nil {
		return out, err
	}
	return out + r.pruneMsg(), nil
}

func (r *Repo) Fetch(token string) (string, error) {
	out, err := r.runGitAuth(token, "fetch", "--all", "--prune")
	if err != nil {
		return out, err
	}
	return out + r.pruneMsg(), nil
}

// pruneMsg удаляет локальные ветки, чей upstream был удалён на сервере
// (виден как "[gone]" после --prune), и возвращает текст для статус-бара.
// Текущую ветку и ветки с неслитыми локальными коммитами не трогает —
// удаление всегда идёт через "git branch -d" (безопасный, не -D).
func (r *Repo) pruneMsg() string {
	deleted, err := r.deleteGoneBranches()
	if err != nil || len(deleted) == 0 {
		return ""
	}
	return "\n" + fmt.Sprintf(i18n.T.PrunedBranchesFmt, strings.Join(deleted, ", "))
}

// deleteGoneBranches удаляет локальные ветки, у которых был upstream, но он
// пропал на remote (обычно потому что ветку удалили на GitHub). Требует,
// чтобы актуальность remote-tracking ссылок уже была обновлена через
// `fetch --prune`/`pull --prune` — сама эта функция сеть не трогает.
func (r *Repo) deleteGoneBranches() ([]string, error) {
	current, err := r.CurrentBranch()
	if err != nil {
		return nil, err
	}
	out, err := runGit(r.Root, "for-each-ref", "--format=%(refname:short)%09%(upstream:track)", "refs/heads")
	if err != nil {
		return nil, err
	}
	var deleted []string
	for _, line := range splitLines(out) {
		name, track, ok := strings.Cut(line, "\t")
		if !ok || name == current || !strings.Contains(track, "[gone]") {
			continue
		}
		if _, err := runGitCombined(r.Root, "branch", "-d", name); err == nil {
			deleted = append(deleted, name)
		}
	}
	return deleted, nil
}

// LsFiles возвращает все отслеживаемые файлы репозитория.
func (r *Repo) LsFiles() ([]string, error) {
	out, err := runGit(r.Root, "ls-files")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// LsFilesOthers возвращает файлы, которые git видит, но не отслеживает, и
// которые не подпадают под существующие правила .gitignore/exclude.
func (r *Repo) LsFilesOthers() ([]string, error) {
	out, err := runGit(r.Root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// LsFilesOthersDirs — то же самое, но полностью неотслеживаемые каталоги
// возвращаются целиком (с завершающим "/"), а не файл за файлом.
func (r *Repo) LsFilesOthersDirs() ([]string, error) {
	out, err := runGit(r.Root, "ls-files", "--others", "--exclude-standard", "--directory")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// Untrack снимает файл с отслеживания, не трогая его на диске (git rm --cached).
func (r *Repo) Untrack(path string) error {
	_, err := runGit(r.Root, "rm", "--cached", "--quiet", "--", path)
	return err
}

// MergeAbort прерывает незавершённый merge, возвращая рабочий каталог и
// индекс к состоянию до его начала.
func (r *Repo) MergeAbort() error {
	_, err := runGitCombined(r.Root, "merge", "--abort")
	return err
}

// RebaseAbort прерывает незавершённый rebase, возвращая ветку к состоянию
// до его начала.
func (r *Repo) RebaseAbort() error {
	_, err := runGitCombined(r.Root, "rebase", "--abort")
	return err
}

// BranchTrack — статус одной локальной ветки относительно её upstream.
type BranchTrack struct {
	Ahead, Behind int
	Gone          bool
	NoUpstream    bool
}

// BranchTracking возвращает статус ahead/behind/gone для всех локальных
// веток одним вызовом for-each-ref — вместо N отдельных rev-list на ветку.
func (r *Repo) BranchTracking() (map[string]BranchTrack, error) {
	out, err := runGit(r.Root, "for-each-ref",
		"--format=%(refname:short)%09%(upstream)%09%(upstream:track)", "refs/heads")
	if err != nil {
		return nil, err
	}
	result := map[string]BranchTrack{}
	for _, line := range splitLines(out) {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		result[parts[0]] = parseTrack(parts[1], parts[2])
	}
	return result, nil
}

// parseTrack разбирает вывод `%(upstream)`/`%(upstream:track)` git
// for-each-ref. upstream пуст только когда у ветки вообще нет upstream —
// этим отличаем "нет upstream" от "полностью синхронна" (у обеих track пуст).
func parseTrack(upstream, track string) BranchTrack {
	if upstream == "" {
		return BranchTrack{NoUpstream: true}
	}
	if strings.Contains(track, "[gone]") {
		return BranchTrack{Gone: true}
	}
	t := BranchTrack{}
	for _, part := range strings.Split(strings.Trim(track, "[]"), ",") {
		part = strings.TrimSpace(part)
		if n, ok := strings.CutPrefix(part, "ahead "); ok {
			t.Ahead, _ = strconv.Atoi(n)
		} else if n, ok := strings.CutPrefix(part, "behind "); ok {
			t.Behind, _ = strconv.Atoi(n)
		}
	}
	return t
}
