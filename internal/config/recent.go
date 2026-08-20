// Package config хранит небольшие пользовательские настройки tgit — не
// секреты (для токена смотри internal/secret), а вещи вроде списка недавно
// открытых репозиториев — в файле конфигурации.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// maxRecent — сколько последних открытых репозиториев хранить; при
// добавлении новой записи старые сверх лимита отбрасываются.
const maxRecent = 15

// RecentEntry — один недавно открытый репозиторий.
type RecentEntry struct {
	Path       string    `json:"path"`
	LastOpened time.Time `json:"last_opened"`
}

func recentPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tgit", "recent.json"), nil
}

// LoadRecent возвращает список недавних репозиториев, от самого свежего.
// Записи, чей путь больше не существует на диске, отфильтровываются.
func LoadRecent() ([]RecentEntry, error) {
	p, err := recentPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []RecentEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, err
	}
	filtered := entries[:0]
	for _, e := range entries {
		if info, err := os.Stat(e.Path); err == nil && info.IsDir() {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}

// AddRecent добавляет путь репозитория в список недавних (или поднимает
// его наверх, если он там уже был) и сохраняет список на диск.
func AddRecent(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	entries, _ := LoadRecent()

	out := []RecentEntry{{Path: abs, LastOpened: time.Now()}}
	for _, e := range entries {
		if e.Path == abs {
			continue
		}
		out = append(out, e)
	}
	if len(out) > maxRecent {
		out = out[:maxRecent]
	}

	p, err := recentPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}
