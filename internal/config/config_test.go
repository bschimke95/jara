package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefault(t *testing.T) {
	cfg := NewDefault()

	if cfg.Jara.RefreshRate != DefaultRefreshRate {
		t.Errorf("default RefreshRate = %v, want %v", cfg.Jara.RefreshRate, DefaultRefreshRate)
	}
	if cfg.Jara.LogLevel != DefaultLogLevel {
		t.Errorf("default LogLevel = %q, want %q", cfg.Jara.LogLevel, DefaultLogLevel)
	}
	if cfg.Jara.Headless {
		t.Error("default Headless should be false")
	}
	if cfg.Jara.Logoless {
		t.Error("default Logoless should be false")
	}
	if cfg.Jara.ReadOnly {
		t.Error("default ReadOnly should be false")
	}
	if cfg.Jara.CharmhubURL != DefaultCharmhubURL {
		t.Errorf("default CharmhubURL = %q, want %q", cfg.Jara.CharmhubURL, DefaultCharmhubURL)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg := NewDefault()
	err := cfg.Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Errorf("Load non-existent file should not error, got: %v", err)
	}
	if cfg.Jara.RefreshRate != DefaultRefreshRate {
		t.Errorf("RefreshRate after missing file = %v, want %v", cfg.Jara.RefreshRate, DefaultRefreshRate)
	}
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `jara:
  refreshRate: 5.0
  logLevel: debug
  headless: true
  readOnly: true
  ui:
    skin:
      primary: "#ff0000"
      highlight: "#00ff00"
      status:
        active: "#aabbcc"
    keys:
      quit:
        keys: ["q", "ctrl+q"]
      help:
        keys: ["h"]
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := NewDefault()
	if err := cfg.Load(cfgPath); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Jara.RefreshRate != 5.0 {
		t.Errorf("RefreshRate = %v, want 5.0", cfg.Jara.RefreshRate)
	}
	if cfg.Jara.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.Jara.LogLevel, "debug")
	}
	if !cfg.Jara.Headless {
		t.Error("Headless should be true")
	}
	if !cfg.Jara.ReadOnly {
		t.Error("ReadOnly should be true")
	}
	if cfg.Jara.UI.Skin.Primary != "#ff0000" {
		t.Errorf("Skin.Primary = %q, want %q", cfg.Jara.UI.Skin.Primary, "#ff0000")
	}
	if cfg.Jara.UI.Skin.Highlight != "#00ff00" {
		t.Errorf("Skin.Highlight = %q, want %q", cfg.Jara.UI.Skin.Highlight, "#00ff00")
	}
	if cfg.Jara.UI.Skin.Status.Active != "#aabbcc" {
		t.Errorf("Skin.Status.Active = %q, want %q", cfg.Jara.UI.Skin.Status.Active, "#aabbcc")
	}
	if cfg.Jara.UI.Keys.Quit == nil {
		t.Fatal("Keys.Quit should not be nil")
	}
	if len(cfg.Jara.UI.Keys.Quit.Keys) != 2 {
		t.Errorf("Keys.Quit.Keys length = %d, want 2", len(cfg.Jara.UI.Keys.Quit.Keys))
	}
	if cfg.Jara.UI.Keys.Help == nil {
		t.Fatal("Keys.Help should not be nil")
	}
	if len(cfg.Jara.UI.Keys.Help.Keys) != 1 || cfg.Jara.UI.Keys.Help.Keys[0] != "h" {
		t.Errorf("Keys.Help.Keys = %v, want [h]", cfg.Jara.UI.Keys.Help.Keys)
	}
	if cfg.Jara.UI.Keys.Back != nil {
		t.Error("Keys.Back should be nil (not set)")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte(":::invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := NewDefault()
	if err := cfg.Load(cfgPath); err == nil {
		t.Error("Load invalid YAML should return error")
	}
}

func TestLoadPreservesDefaultsForZeroValues(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `jara:
  headless: true
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := NewDefault()
	if err := cfg.Load(cfgPath); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Jara.RefreshRate != DefaultRefreshRate {
		t.Errorf("RefreshRate = %v, want %v (default preserved)", cfg.Jara.RefreshRate, DefaultRefreshRate)
	}
	if cfg.Jara.LogLevel != DefaultLogLevel {
		t.Errorf("LogLevel = %q, want %q (default preserved)", cfg.Jara.LogLevel, DefaultLogLevel)
	}
	if cfg.Jara.CharmhubURL != DefaultCharmhubURL {
		t.Errorf("CharmhubURL = %q, want %q (default preserved)", cfg.Jara.CharmhubURL, DefaultCharmhubURL)
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subdir", "config.yaml")

	cfg := NewDefault()
	cfg.Jara.RefreshRate = 10
	cfg.Jara.LogLevel = "error"
	cfg.Jara.UI.Skin.Primary = "#123456"

	if err := cfg.Save(cfgPath); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded := NewDefault()
	if err := loaded.Load(cfgPath); err != nil {
		t.Fatalf("Load after Save returned error: %v", err)
	}

	if loaded.Jara.RefreshRate != 10 {
		t.Errorf("roundtrip RefreshRate = %v, want 10", loaded.Jara.RefreshRate)
	}
	if loaded.Jara.LogLevel != "error" {
		t.Errorf("roundtrip LogLevel = %q, want %q", loaded.Jara.LogLevel, "error")
	}
	if loaded.Jara.UI.Skin.Primary != "#123456" {
		t.Errorf("roundtrip Skin.Primary = %q, want %q", loaded.Jara.UI.Skin.Primary, "#123456")
	}
}

func TestRefreshDuration(t *testing.T) {
	cfg := NewDefault()
	d := cfg.RefreshDuration()
	if d.Seconds() != DefaultRefreshRate {
		t.Errorf("RefreshDuration = %v, want %vs", d, DefaultRefreshRate)
	}

	cfg.Jara.RefreshRate = 10
	d = cfg.RefreshDuration()
	if d.Seconds() != 10 {
		t.Errorf("RefreshDuration = %v, want 10s", d)
	}

	cfg.Jara.RefreshRate = 0
	d = cfg.RefreshDuration()
	if d.Seconds() != DefaultRefreshRate {
		t.Errorf("RefreshDuration with 0 = %v, want %vs", d, DefaultRefreshRate)
	}
}

func TestOverride(t *testing.T) {
	cfg := NewDefault()

	logLevel := "warn"
	headless := true

	flags := &Flags{
		LogLevel: &logLevel,
		Headless: &headless,
	}

	cfg.Override(flags)

	if cfg.Jara.LogLevel != "warn" {
		t.Errorf("after Override, LogLevel = %q, want %q", cfg.Jara.LogLevel, "warn")
	}
	if !cfg.Jara.Headless {
		t.Error("after Override, Headless should be true")
	}
}

func TestOverrideNilFlags(t *testing.T) {
	cfg := NewDefault()
	cfg.Override(nil)

	if cfg.Jara.LogLevel != DefaultLogLevel {
		t.Errorf("LogLevel changed with nil flags: %v", cfg.Jara.LogLevel)
	}
}
