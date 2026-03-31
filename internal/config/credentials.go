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
//	copilot: GITHUB_TOKEN env → gh CLI hosts.yml (github.com oauth_token)
//	gemini:  GOOGLE_AI_API_KEY env → GEMINI_API_KEY env
func LoadAICredential(provider string) string {
	switch strings.ToLower(provider) {
	case "copilot", "":
		if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
			return tok
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

// ghCLIToken attempts to read a GitHub token from the gh CLI config file
// at ~/.config/gh/hosts.yml.
func ghCLIToken() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(home + "/.config/gh/hosts.yml")
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
