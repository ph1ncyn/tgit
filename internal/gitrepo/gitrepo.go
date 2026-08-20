// Package gitrepo — тонкая кроссплатформенная обёртка над системным git.
// Полагается на установленный git в PATH: это надёжнее и совместимее с
// поведением реального git, чем переизобретать формат объектов через
// отдельную библиотеку.
package gitrepo

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// FileStatus — одна строка `git status --porcelain`.
type FileStatus struct {
	X, Y byte
	Path string
}

// Repo — открытый git-репозиторий.
type Repo struct {
	Root string
}

// Open находит корень git-репозитория, начиная с dir.
func Open(dir string) (*Repo, error) {
	out, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("не git-репозиторий: %w", err)
	}
	return &Repo{Root: strings.TrimSpace(out)}, nil
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

// CurrentBranch возвращает имя текущей ветки (или "HEAD" при detached HEAD).
func (r *Repo) CurrentBranch() (string, error) {
	out, err := runGit(r.Root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Branches возвращает список локальных веток.
func (r *Repo) Branches() ([]string, error) {
	out, err := runGit(r.Root, "branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, l := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(l) != "" {
			branches = append(branches, l)
		}
	}
	return branches, nil
}

// Status возвращает список изменённых/новых файлов.
func (r *Repo) Status() ([]FileStatus, error) {
	out, err := runGit(r.Root, "status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}
	var files []FileStatus
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
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
