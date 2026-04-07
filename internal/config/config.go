package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	DefaultBaseURL = "https://www.nanoinfluencer.ai"
	ConfigDirEnv   = "NANOINF_CONFIG_DIR"
)

var ErrTokenNotConfigured = errors.New("access token not configured")

type Config struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token,omitempty"`
}

func Default() Config {
	return Config{
		BaseURL: DefaultBaseURL,
	}
}

func ResolveDir() (string, error) {
	if dir := os.Getenv(ConfigDirEnv); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "nanoinf"), nil
}

func ResolvePath() (string, error) {
	dir, err := ResolveDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (Config, error) {
	path, err := ResolvePath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, err
	}

	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	return cfg, nil
}

func Save(cfg Config) error {
	path, err := ResolvePath()
	if err != nil {
		return err
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func PreviewToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 12 {
		return token
	}
	return token[:8] + "..." + token[len(token)-8:]
}
