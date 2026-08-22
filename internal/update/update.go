// Package update сравнивает установленную версию tgit с версией,
// опубликованной в version.json на ветке main репозитория на GitHub, и
// умеет обновить себя на месте: git pull в исходниках + go build + замена
// установленного бинарника.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tgit/internal/i18n"
)

// remoteVersionURL — raw-файл version.json на ветке main. Не требует токена
// (репозиторий публичный) и не завязан на git tags/releases.
const remoteVersionURL = "https://raw.githubusercontent.com/ph1ncyn/tgit/main/version.json"

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Local — содержимое local_version.json, которое install.sh/install.ps1
// кладут рядом с установленным бинарником.
type Local struct {
	Version   string `json:"version"`
	SourceDir string `json:"source_dir"`
}

// LoadLocal ищет local_version.json рядом с текущим исполняемым файлом.
// Отсутствие файла не ошибка (например, `go run` в разработке, или сборка
// без install.sh) — просто ok=false, и проверка обновлений тихо выключается.
func LoadLocal() (local Local, ok bool) {
	exe, err := exePath()
	if err != nil {
		return Local{}, false
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "local_version.json"))
	if err != nil {
		return Local{}, false
	}
	if err := json.Unmarshal(data, &local); err != nil || local.Version == "" {
		return Local{}, false
	}
	return local, true
}

// exePathOverride подменяет результат exePath() в тестах — сам процесс
// тестов запущен из go test, а не из установленного бинарника, который
// нужно проверить на подмену.
var exePathOverride string

// exePath возвращает путь к текущему исполняемому файлу, разрешая симлинки
// (важно для ~/.local/bin, куда install.sh часто кладёт файл напрямую, но
// путь запуска может прийти через симлинк из другого каталога PATH).
func exePath() (string, error) {
	if exePathOverride != "" {
		return exePathOverride, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved, nil
	}
	return exe, nil
}

// FetchRemoteVersion скачивает version.json из ветки main на GitHub и
// возвращает поле version.
func FetchRemoteVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteVersionURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf(i18n.T.NetworkErrFmt, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(i18n.T.UnexpectedStatusFmt, resp.StatusCode)
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", fmt.Errorf(i18n.T.ParseFailedFmt, err)
	}
	return v.Version, nil
}

// Newer сообщает, действительно ли remote новее local. Версии в формате
// YY.MM.DD (например "26.08.22") с ведущими нулями сравниваются
// лексикографически — для дат такой фиксированной ширины это даёт верный
// хронологический порядок без парсинга. Так индикатор обновления не
// загорается зря, если у remote вдруг более старая версия (откат релиза).
func Newer(local, remote string) bool {
	return remote != "" && remote > local
}

// Apply обновляет tgit на месте: git pull в SourceDir, сборка нового
// бинарника и атомарная замена текущего исполняемого файла. Возвращает
// новую версию и путь к бинарнику (для последующего перезапуска) либо
// ошибку. На Windows замена запущенного .exe штатно недоступна (файл
// заблокирован ОС) — Apply в этом случае сразу возвращает понятную ошибку,
// не пытаясь молча испортить установку.
func Apply(ctx context.Context, local Local) (newVersion, exe string, err error) {
	if runtime.GOOS == "windows" {
		return "", "", fmt.Errorf("%s", i18n.T.UpdateWindowsUnsupportedMsg)
	}
	if local.SourceDir == "" {
		return "", "", fmt.Errorf("%s", i18n.T.UpdateNoSourceDirMsg)
	}
	if _, err := os.Stat(local.SourceDir); err != nil {
		return "", "", fmt.Errorf(i18n.T.UpdateSourceDirMissingFmt, local.SourceDir)
	}

	if out, err := runCmd(local.SourceDir, "git", "pull", "--ff-only"); err != nil {
		return "", "", fmt.Errorf("git pull: %w\n%s", err, out)
	}

	versionData, err := os.ReadFile(filepath.Join(local.SourceDir, "version.json"))
	if err != nil {
		return "", "", fmt.Errorf(i18n.T.UpdateReadVersionFailedFmt, err)
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(versionData, &v); err != nil || v.Version == "" {
		return "", "", fmt.Errorf(i18n.T.UpdateReadVersionFailedFmt, err)
	}

	exe, err = exePath()
	if err != nil {
		return "", "", err
	}

	tmpBin := exe + ".update-tmp"
	buildArgs := []string{"build", "-ldflags", "-X main.version=" + v.Version, "-o", tmpBin, "."}
	if out, err := runCmd(local.SourceDir, "go", buildArgs...); err != nil {
		_ = os.Remove(tmpBin)
		return "", "", fmt.Errorf("go build: %w\n%s", err, out)
	}
	if err := os.Chmod(tmpBin, 0o755); err != nil {
		_ = os.Remove(tmpBin)
		return "", "", err
	}

	if err := os.Rename(tmpBin, exe); err != nil {
		_ = os.Remove(tmpBin)
		return "", "", err
	}

	newLocal := Local{Version: v.Version, SourceDir: local.SourceDir}
	newLocalData, _ := json.MarshalIndent(newLocal, "", "  ")
	_ = os.WriteFile(filepath.Join(filepath.Dir(exe), "local_version.json"), newLocalData, 0o644)

	return v.Version, exe, nil
}

// Restart заменяет текущий процесс на свежеустановленный бинарник exe (см.
// restart в restart_unix.go/restart_windows.go). При успехе не возвращается
// вообще — процесс уже стал новой программой. Вызывающий код должен сам
// корректно завершить bubbletea (tea.Quit) ДО вызова Restart, чтобы
// терминал успел вернуться в обычный режим — иначе новый образ процесса
// войдёт в raw mode поверх ещё не восстановленного состояния терминала.
func Restart(exe string) error {
	return restart(exe)
}

func runCmd(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
