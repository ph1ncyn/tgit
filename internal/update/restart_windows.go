//go:build windows

package update

import "fmt"

// restart на Windows недостижим в реальной работе: Apply уже отказывает
// раньше с понятной ошибкой (см. i18n.T.UpdateWindowsUnsupportedMsg), так
// что RestartExe никогда не заполняется на этой ОС. Заглушка нужна только
// для компиляции — install.ps1 тоже собирает этот пакет.
func restart(exe string) error {
	return fmt.Errorf("restart is not supported on windows")
}
