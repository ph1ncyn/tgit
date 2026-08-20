package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Hyperlink оборачивает текст в OSC8-гиперссылку, но только если терминал по
// известным признакам её поддерживает — иначе многие терминалы (в частности
// многие эмуляторы на Linux) не интерпретируют escape-последовательность и
// показывают её как мусорный текст вокруг ссылки. В неизвестном терминале
// возвращаем просто label (полный URL), это надёжнее.
func Hyperlink(url, label string) string {
	if !terminalSupportsHyperlinks() {
		return label
	}
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, label)
}

func terminalSupportsHyperlinks() bool {
	if os.Getenv("TGIT_FORCE_HYPERLINKS") == "1" {
		return true
	}

	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "vscode", "Hyper", "Tabby", "ghostty":
		return true
	}

	if os.Getenv("WT_SESSION") != "" { // Windows Terminal
		return true
	}
	if os.Getenv("KONSOLE_VERSION") != "" { // Konsole (KDE)
		return true
	}

	term := os.Getenv("TERM")
	if strings.Contains(term, "kitty") || strings.Contains(term, "alacritty") {
		return true
	}

	// VTE-based terminals (GNOME Terminal, Tilix, XFCE Terminal, ...) добавили
	// поддержку OSC8 начиная с VTE 0.50 (версия хранится как MMmmpp, напр. 6003 = 0.60.3).
	if v := os.Getenv("VTE_VERSION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 5000 {
			return true
		}
	}

	return false
}
