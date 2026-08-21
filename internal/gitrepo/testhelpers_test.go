package gitrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runGitT выполняет git-команду в dir и завершает тест при ошибке.
func runGitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// newTestRepo создаёт пустой (без коммитов) локальный репозиторий с
// настроенным user.email/user.name, без remote.
func newTestRepo(t *testing.T) *Repo {
	t.Helper()
	dir := t.TempDir()
	runGitT(t, dir, "init", "-q")
	runGitT(t, dir, "config", "user.email", "test@example.com")
	runGitT(t, dir, "config", "user.name", "Test")
	return &Repo{Root: dir}
}

// writeAndCommit создаёт/перезаписывает файл path и коммитит его.
func writeAndCommit(t *testing.T, repo *Repo, path, content, message string) {
	t.Helper()
	full := filepath.Join(repo.Root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitT(t, repo.Root, "add", path)
	runGitT(t, repo.Root, "commit", "-q", "-m", message)
}

// newRepoWithRemote создаёт bare-репозиторий как "remote" и клон с
// настроенным user.email/user.name, отслеживающий его через origin — та же
// фикстура, что использовалась вручную для проверки gone-branch/ahead-behind
// сценариев в этой сессии.
func newRepoWithRemote(t *testing.T) (local *Repo, remoteDir string) {
	t.Helper()
	remoteDir = t.TempDir()
	runGitT(t, remoteDir, "init", "--bare", "-q")

	cloneDir := t.TempDir()
	runGitT(t, ".", "clone", "-q", remoteDir, cloneDir)
	runGitT(t, cloneDir, "config", "user.email", "test@example.com")
	runGitT(t, cloneDir, "config", "user.name", "Test")

	return &Repo{Root: cloneDir}, remoteDir
}

// cloneRemote делает второй клон того же bare-remote — имитирует другого
// разработчика/устройство, пушащего изменения независимо от local.
func cloneRemote(t *testing.T, remoteDir string) *Repo {
	t.Helper()
	dir := t.TempDir()
	runGitT(t, ".", "clone", "-q", remoteDir, dir)
	runGitT(t, dir, "config", "user.email", "test@example.com")
	runGitT(t, dir, "config", "user.name", "Test")
	return &Repo{Root: dir}
}
