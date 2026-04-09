package config

import (
	"os"
	"path/filepath"
)

// DefaultConfigDir returns the config directory for jara.
// It respects $JARA_CONFIG_DIR, otherwise defaults to ~/.jara.
func DefaultConfigDir() string {
	if d := os.Getenv("JARA_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".jara")
}

// DefaultConfigFile returns the path to the default config file.
func DefaultConfigFile() string {
	return filepath.Join(DefaultConfigDir(), "config.yaml")
}

// DefaultLogDir returns the XDG-compliant log directory.
func DefaultLogDir() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	return filepath.Join(cacheDir, AppName)
}

// DefaultLogFile returns the path to the default log file.
func DefaultLogFile() string {
	return filepath.Join(DefaultLogDir(), "jara.log")
}

// DefaultSkinDir returns the path to skin/theme files.
func DefaultSkinDir() string {
	return filepath.Join(DefaultConfigDir(), "skins")
}
