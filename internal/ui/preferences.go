package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Preferences は永続化されるユーザー設定。
type Preferences struct {
	ThemeName string `json:"theme"`
}

// DefaultPreferences は初期値のついた Preferences を返す。
func DefaultPreferences() *Preferences {
	return &Preferences{ThemeName: ThemeBlack.Name}
}

func preferencesPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "space-miner", "settings.json"), nil
}

// LoadPreferences は設定ファイルを読み込む。失敗時はデフォルト値を返す。
func LoadPreferences() *Preferences {
	path, err := preferencesPath()
	if err != nil {
		return DefaultPreferences()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultPreferences()
	}
	var p Preferences
	if err := json.Unmarshal(data, &p); err != nil {
		return DefaultPreferences()
	}
	return &p
}

// SavePreferences は設定ファイルに書き出す。
func SavePreferences(p *Preferences) error {
	path, err := preferencesPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
