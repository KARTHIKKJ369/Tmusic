package config

import (
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.MusicDir = "/test/music"
	cfg.Volume = 0.65
	cfg.Shuffle = true
	cfg.Repeat = RepeatTrack

	if err := Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.MusicDir != cfg.MusicDir {
		t.Errorf("expected MusicDir %q, got %q", cfg.MusicDir, loaded.MusicDir)
	}
	if loaded.Volume != cfg.Volume {
		t.Errorf("expected Volume %f, got %f", cfg.Volume, loaded.Volume)
	}
	if loaded.Shuffle != cfg.Shuffle {
		t.Errorf("expected Shuffle %v, got %v", cfg.Shuffle, loaded.Shuffle)
	}
	if loaded.Repeat != cfg.Repeat {
		t.Errorf("expected Repeat %v, got %v", cfg.Repeat, loaded.Repeat)
	}
}

func TestJSONState(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	type Data struct {
		Items []string `json:"items"`
	}

	d := Data{Items: []string{"a", "b", "c"}}
	if err := SaveJSON("test.json", d); err != nil {
		t.Fatalf("failed to save JSON: %v", err)
	}

	var loaded Data
	if err := LoadJSON("test.json", &loaded); err != nil {
		t.Fatalf("failed to load JSON: %v", err)
	}

	if len(loaded.Items) != 3 || loaded.Items[1] != "b" {
		t.Fatalf("unexpected loaded items: %v", loaded.Items)
	}
}
