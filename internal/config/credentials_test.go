package config

import (
	"testing"
)

func TestLoadAICredentialCopilotEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    string
	}{
		{
			name:    "GITHUB_TOKEN takes priority",
			envVars: map[string]string{"GITHUB_TOKEN": "gh-tok", "GH_TOKEN": "alt-tok"},
			want:    "gh-tok",
		},
		{
			name:    "GH_TOKEN used when GITHUB_TOKEN empty",
			envVars: map[string]string{"GH_TOKEN": "alt-tok"},
			want:    "alt-tok",
		},
		{
			name:    "COPILOT_GITHUB_TOKEN used as fallback",
			envVars: map[string]string{"COPILOT_GITHUB_TOKEN": "cp-tok"},
			want:    "cp-tok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all copilot-related env vars.
			for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN", "COPILOT_GITHUB_TOKEN"} {
				t.Setenv(k, "")
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			got := LoadAICredential("copilot")
			if got != tt.want {
				t.Errorf("LoadAICredential(copilot) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadAICredentialUnknownProvider(t *testing.T) {
	got := LoadAICredential("openai")
	if got != "" {
		t.Errorf("LoadAICredential(openai) = %q, want empty", got)
	}
}
