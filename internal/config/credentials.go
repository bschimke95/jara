package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadAICredential returns the API key/token for the given provider.
// It checks environment variables first, then falls back to the gh CLI
// config for the "copilot" provider.
//
// Lookup order:
//
//	copilot: GITHUB_TOKEN → GH_TOKEN → COPILOT_GITHUB_TOKEN → gh CLI hosts.yml
//	gemini:  GOOGLE_AI_API_KEY env → GEMINI_API_KEY env
func LoadAICredential(provider string) string {
	switch strings.ToLower(provider) {
	case "copilot", "":
		for _, env := range []string{"GITHUB_TOKEN", "GH_TOKEN", "COPILOT_GITHUB_TOKEN"} {
			if tok := os.Getenv(env); tok != "" {
				return tok
			}
		}
		return ghCLIToken()
	case "gemini":
		if k := os.Getenv("GOOGLE_AI_API_KEY"); k != "" {
			return k
		}
		return os.Getenv("GEMINI_API_KEY")
	default:
		return ""
	}
}

// ghCLIToken attempts to read a GitHub token from the gh CLI config file.
// It respects GH_CONFIG_DIR and XDG_CONFIG_HOME, falling back to ~/.config/gh.
func ghCLIToken() string {
	dir := os.Getenv("GH_CONFIG_DIR")
	if dir == "" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			dir = xdg + "/gh"
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			dir = home + "/.config/gh"
		}
	}
	data, err := os.ReadFile(dir + "/hosts.yml")
	if err != nil {
		return ""
	}

	var hosts map[string]struct {
		OAuthToken string `yaml:"oauth_token"`
	}
	if err := yaml.Unmarshal(data, &hosts); err != nil {
		return ""
	}
	if gh, ok := hosts["github.com"]; ok {
		return gh.OAuthToken
	}
	return ""
}
