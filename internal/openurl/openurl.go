// Package openurl открывает ссылку в браузере по умолчанию для текущей ОС.
package openurl

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Open запускает системный обработчик URL. Используется как запасной вариант
// для терминалов, не поддерживающих кликабельные OSC8-гиперссылки.
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("don't know how to open a browser on %s", runtime.GOOS)
	}
	return cmd.Start()
}
