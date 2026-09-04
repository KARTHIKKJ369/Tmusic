// Package config manages persistent player configuration and state.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const appName = "muse"

// RepeatMode controls repeat behaviour.
type RepeatMode string

const (
	RepeatOff   RepeatMode = "off"
	RepeatTrack RepeatMode = "track"
	RepeatQueue RepeatMode = "queue"
)

// Config is the user-facing TOML configuration.
type Config struct {
	MusicDir string     `toml:"music_dir"`
	Volume   float64    `toml:"volume"`
	Shuffle  bool       `toml:"shuffle"`
	Repeat   RepeatMode `toml:"repeat"`
	Theme    string     `toml:"theme"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Volume:  0.8,
		Shuffle: false,
		Repeat:  RepeatOff,
		Theme:   "dark",
	}
}

// Dir returns the app config directory, creating it if needed.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, appName)
	return dir, os.MkdirAll(dir, 0o755)
}

// Load reads config from disk, returning defaults if the file doesn't exist.
func Load() (Config, error) {
	cfg := DefaultConfig()
	dir, err := Dir()
	if err != nil {
		return cfg, err
	}
	path := filepath.Join(dir, "config.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	_, err = toml.DecodeFile(path, &cfg)
	return cfg, err
}

// Save writes config to disk.
func Save(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "config.toml"))
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// LoadJSON unmarshals a JSON state file from the config dir.
func LoadJSON(filename string, v any) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// SaveJSON marshals v and writes it to a JSON state file in the config dir.
func SaveJSON(filename string, v any) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), data, 0o644)
}
