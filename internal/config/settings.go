package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// settings — небольшие настройки tgit, не связанные со списком недавних
// репозиториев (тот хранится отдельно, в recent.json — см. recent.go),
// поэтому отдельный файл, а не смешивание форматов в одном.
type settings struct {
	// Language — код языка интерфейса ("en"/"ru", см. i18n.Lang.Code), выбранный
	// на экране выбора языка. Пусто, если пользователь ещё ни разу не выбирал —
	// тогда экран выбора показывается снова при каждом запуске.
	Language string `json:"language,omitempty"`
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tgit", "settings.json"), nil
}

func loadSettings() (settings, error) {
	p, err := settingsPath()
	if err != nil {
		return settings{}, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return settings{}, nil
		}
		return settings{}, err
	}
	var s settings
	if err := json.Unmarshal(b, &s); err != nil {
		return settings{}, err
	}
	return s, nil
}

func saveSettings(s settings) error {
	p, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

// LoadLanguage returns the previously saved interface language code and true,
// or ("", false) if none was saved yet (or it couldn't be read).
func LoadLanguage() (string, bool) {
	s, err := loadSettings()
	if err != nil || s.Language == "" {
		return "", false
	}
	return s.Language, true
}

// SaveLanguage persists the chosen interface language code, keeping any other
// saved settings untouched.
func SaveLanguage(code string) error {
	s, _ := loadSettings()
	s.Language = code
	return saveSettings(s)
}
