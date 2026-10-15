package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const Version = 1

type Config struct {
	Version int    `json:"version"`
	URL     string `json:"url"`
	Token   string `json:"token"`
}

// Path returns the config file location, in precedence order:
// POSTIER_CONFIG, then $XDG_CONFIG_HOME/postier, then ~/.config/postier.
// It does not touch the filesystem: the file may legitimately not exist yet.
func Path() (string, error) {
	if val, ok := os.LookupEnv("POSTIER_CONFIG"); ok && val != "" {
		return val, nil
	}

	if val, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok && val != "" {
		return filepath.Join(val, "postier", "config.json"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate home directory: %w", err)
	}

	return filepath.Join(home, ".config", "postier", "config.json"), nil
}

// Load reads the config file. A missing file is NOT an error — it returns an
// empty Config so that `pst connect` works on a fresh install. A corrupt file
// IS an error, and the message names the path.
func Load() (*Config, error) {
	configPath, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if errors.Is(err, fs.ErrNotExist) {
		return &Config{Version: Version}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read config file %s: %w", configPath, err)
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config file %s is malformed: %w", configPath, err)
	}

	return &c, nil
}

// Save writes the config file 0600, inside a directory created 0700.
func (c *Config) Save() error {
	configPath, err := Path()
	if err != nil {
		return err
	}

	// Unconditional: MkdirAll is a no-op when the directory already exists.
	// 0700 because the file below holds a bearer token.
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	c.Version = Version

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("cannot write config file %s: %w", configPath, err)
	}

	return nil
}

// Resolve applies flag > env > file > built-in default, treating "" as unset at
// every layer. Pure apart from os.Getenv, so tests can cover the precedence
// matrix without touching the filesystem.
func Resolve(file *Config, flagURL, flagToken, buildDefault string) *Config {
	// A nil file config means the same thing as an empty one: nothing on disk.
	if file == nil {
		file = &Config{}
	}

	out := *file // value copy: safe while every field is a value type

	if out.URL == "" {
		out.URL = buildDefault
	}

	if val := os.Getenv("POSTIER_URL"); val != "" {
		out.URL = val
	}
	if val := os.Getenv("POSTIER_TOKEN"); val != "" {
		out.Token = val
	}

	if flagURL != "" {
		out.URL = flagURL
	}
	if flagToken != "" {
		out.Token = flagToken
	}

	return &out
}
