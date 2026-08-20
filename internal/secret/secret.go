// Package secret хранит GitHub-токен кроссплатформенно: в системном хранилище
// секретов (Keychain на macOS, Credential Manager на Windows, Secret Service/dbus
// на Linux), а если оно недоступно (например headless Linux без dbus) — в файле
// конфигурации с правами 0600.
package secret

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	service = "tgit"
	account = "github-token"
)

// Save сохраняет токен, предпочитая системное хранилище секретов.
func Save(token string) error {
	if err := keyring.Set(service, account, token); err == nil {
		return nil
	}
	return saveFile(token)
}

// Load возвращает сохранённый токен, если он есть.
func Load() (string, error) {
	if tok, err := keyring.Get(service, account); err == nil && tok != "" {
		return tok, nil
	}
	return loadFile()
}

// Delete удаляет сохранённый токен из всех мест хранения.
func Delete() error {
	_ = keyring.Delete(service, account)
	return deleteFile()
}

func fallbackPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tgit", "github_token"), nil
}

func saveFile(token string) error {
	p, err := fallbackPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(token), 0o600)
}

func loadFile() (string, error) {
	p, err := fallbackPath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func deleteFile() error {
	p, err := fallbackPath()
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
