package config

import (
	"os"
	"testing"
)

func TestDefaultConfigDir(t *testing.T) {
	// Should return a non-empty path.
	dir := DefaultConfigDir()
	if dir == "" {
		t.Error("DefaultConfigDir returned empty string")
	}
}

func TestDefaultConfigDirRespectsEnv(t *testing.T) {
	t.Setenv("JARA_CONFIG_DIR", "/tmp/jara-test-config")
	dir := DefaultConfigDir()
	if dir != "/tmp/jara-test-config" {
		t.Errorf("DefaultConfigDir with JARA_CONFIG_DIR = %q, want /tmp/jara-test-config", dir)
	}
}

func TestDefaultConfigDirRespectsXDG(t *testing.T) {
	t.Setenv("JARA_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	dir := DefaultConfigDir()
	if dir != "/tmp/xdg-test/jara" {
		t.Errorf("DefaultConfigDir with XDG = %q, want /tmp/xdg-test/jara", dir)
	}
}

func TestDefaultConfigFile(t *testing.T) {
	t.Setenv("JARA_CONFIG_DIR", "/tmp/jara-test-config")
	f := DefaultConfigFile()
	if f != "/tmp/jara-test-config/config.yaml" {
		t.Errorf("DefaultConfigFile = %q, want /tmp/jara-test-config/config.yaml", f)
	}
}

func TestDefaultLogFile(t *testing.T) {
	f := DefaultLogFile()
	if f == "" {
		t.Error("DefaultLogFile returned empty string")
	}
}

func TestDefaultSkinDir(t *testing.T) {
	t.Setenv("JARA_CONFIG_DIR", "/tmp/jara-test-config")
	d := DefaultSkinDir()
	if d != "/tmp/jara-test-config/skins" {
		t.Errorf("DefaultSkinDir = %q, want /tmp/jara-test-config/skins", d)
	}
}

func TestDefaultLogDir(t *testing.T) {
	d := DefaultLogDir()
	if d == "" {
		t.Error("DefaultLogDir returned empty string")
	}
	// Should end with /jara.
	if len(d) < 5 || d[len(d)-5:] != "/jara" {
		t.Errorf("DefaultLogDir = %q, should end with /jara", d)
	}
}

func TestDefaultConfigDirFallbackToHome(t *testing.T) {
	t.Setenv("JARA_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	dir := DefaultConfigDir()
	home, _ := os.UserHomeDir()
	expected := home + "/.config/jara"
	if dir != expected {
		t.Errorf("DefaultConfigDir fallback = %q, want %q", dir, expected)
	}
}
