// Package config provides configuration management for jara.
// Configuration is loaded from a YAML file (typically ~/.config/jara/config.yaml)
// and can be overridden by CLI flags. The design follows k9s conventions:
// XDG-compliant paths, layered overrides, and a skin/theme system.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// AppName is the application name used for directory paths.
	AppName = "jara"

	// DefaultRefreshRate is the default status poll interval in seconds.
	DefaultRefreshRate = 3.0

	// DefaultLogLevel is the default log level.
	DefaultLogLevel = "info"

	// DefaultCharmhubURL is the default Charmhub API base URL used for
	// autocomplete suggestions in deploy modal.
	DefaultCharmhubURL = "https://api.charmhub.io"

	// DefaultToastDuration is the default duration for error toast messages.
	DefaultToastDuration = 4 * time.Second
)

// Config is the top-level configuration for jara.
type Config struct {
	Jara JaraConfig `yaml:"jara"`
}

// JaraConfig holds all jara-specific settings.
type JaraConfig struct {
	// RefreshRate is the status poll interval in seconds.
	RefreshRate float64 `yaml:"refreshRate,omitempty"`

	// LogLevel controls the verbosity of logging (error, warn, info, debug).
	LogLevel string `yaml:"logLevel,omitempty"`

	// LogFile overrides the default log file path.
	LogFile string `yaml:"logFile,omitempty"`

	// Headless hides the header when true.
	Headless bool `yaml:"headless,omitempty"`

	// Logoless hides the logo in the header when true.
	Logoless bool `yaml:"logoless,omitempty"`

	// ReadOnly disables write operations when true.
	ReadOnly bool `yaml:"readOnly,omitempty"`

	// CharmhubURL is the base URL for Charmhub API requests.
	CharmhubURL string `yaml:"charmhubURL,omitempty"`

	// AI holds the LLM provider configuration for the AI chat view.
	AI AIConfig `yaml:"ai,omitempty"`

	// UI holds the theme and key binding configuration.
	UI UIConfig `yaml:"ui,omitempty"`

	// ToastDuration controls how long error toasts are displayed.
	// Accepts Go duration strings (e.g. "4s", "2500ms"). Defaults to 4s.
	ToastDuration time.Duration `yaml:"toastDuration,omitempty"`
}

// AIConfig holds configuration for the LLM-powered analysis chat.
type AIConfig struct {
	// Provider selects the LLM backend: "copilot" or "gemini".
	Provider string `yaml:"provider,omitempty"`

	// Model overrides the default model for the chosen provider.
	Model string `yaml:"model,omitempty"`

	// BaseURL overrides the API endpoint (useful for proxies).
	BaseURL string `yaml:"baseURL,omitempty"`

	// SystemPrompt replaces the built-in system prompt.
	SystemPrompt string `yaml:"systemPrompt,omitempty"`

	// Temperature controls sampling randomness (0.0–2.0).
	Temperature *float64 `yaml:"temperature,omitempty"`

	// MaxTokens limits the maximum response length.
	MaxTokens *int `yaml:"maxTokens,omitempty"`
}

// UIConfig groups all user-interface related configuration.
type UIConfig struct {
	// Skin holds the color theme configuration.
	Skin SkinConfig `yaml:"skin,omitempty"`

	// Keys holds the key binding overrides.
	Keys KeysConfig `yaml:"keys,omitempty"`
}

// SkinConfig defines the color theme. All fields are optional hex color
// strings (e.g. "#00bfff"). Empty strings fall back to the compiled default.
type SkinConfig struct {
	// LogoColor is the accent color for the ASCII logo.
	LogoColor string `yaml:"logoColor,omitempty"`

	// Primary is used for table headers, active crumbs, and key hints.
	Primary string `yaml:"primary,omitempty"`

	// Secondary is for muted informational text.
	Secondary string `yaml:"secondary,omitempty"`

	// Title is the default text color.
	Title string `yaml:"title,omitempty"`

	// Subtle is used for borders and separators.
	Subtle string `yaml:"subtle,omitempty"`

	// Highlight is the selected-row background color.
	Highlight string `yaml:"highlight,omitempty"`

	// Muted is for less-important text.
	Muted string `yaml:"muted,omitempty"`

	// HintKey is the color for key hint brackets and keys.
	HintKey string `yaml:"hintKey,omitempty"`

	// HintDesc is the color for key hint descriptions.
	HintDesc string `yaml:"hintDesc,omitempty"`

	// CrumbFg is text color inside crumb indicators.
	CrumbFg string `yaml:"crumbFg,omitempty"`

	// CrumbBg is the crumb background color.
	CrumbBg string `yaml:"crumbBg,omitempty"`

	// Border is the color for box borders.
	Border string `yaml:"border,omitempty"`

	// BorderTitle is the color for title text in borders.
	BorderTitle string `yaml:"borderTitle,omitempty"`

	// InfoLabel is the color for dim labels (e.g. "Controller:").
	InfoLabel string `yaml:"infoLabel,omitempty"`

	// InfoValue is the color for values next to info labels.
	InfoValue string `yaml:"infoValue,omitempty"`

	// Error is the color for error messages.
	Error string `yaml:"error,omitempty"`

	// SearchHighlightFg is the foreground for search-match highlighting.
	SearchHighlightFg string `yaml:"searchHighlightFg,omitempty"`

	// SearchHighlightBg is the background for search-match highlighting.
	SearchHighlightBg string `yaml:"searchHighlightBg,omitempty"`

	// CrumbBgAlt is the background for secondary/context crumbs.
	CrumbBgAlt string `yaml:"crumbBgAlt,omitempty"`

	// CheckGreen is the color for positive check marks.
	CheckGreen string `yaml:"checkGreen,omitempty"`

	// CheckRed is the color for negative/unchecked marks.
	CheckRed string `yaml:"checkRed,omitempty"`

	// AssistantLabel is the color for the "Assistant:" label in chat.
	AssistantLabel string `yaml:"assistantLabel,omitempty"`

	// Status maps Juju status strings to hex colors for overrides.
	Status StatusColorsConfig `yaml:"status,omitempty"`
}

// StatusColorsConfig allows overriding the color used for each Juju status.
type StatusColorsConfig struct {
	Active      string `yaml:"active,omitempty"`
	Idle        string `yaml:"idle,omitempty"`
	Running     string `yaml:"running,omitempty"`
	Started     string `yaml:"started,omitempty"`
	Blocked     string `yaml:"blocked,omitempty"`
	Error       string `yaml:"error,omitempty"`
	Lost        string `yaml:"lost,omitempty"`
	Down        string `yaml:"down,omitempty"`
	Waiting     string `yaml:"waiting,omitempty"`
	Allocating  string `yaml:"allocating,omitempty"`
	Pending     string `yaml:"pending,omitempty"`
	Maintenance string `yaml:"maintenance,omitempty"`
	Executing   string `yaml:"executing,omitempty"`
	Terminated  string `yaml:"terminated,omitempty"`
	Unknown     string `yaml:"unknown,omitempty"`
	Stopped     string `yaml:"stopped,omitempty"`
	Default     string `yaml:"default,omitempty"`
}

// KeyBindingConfig defines a single key binding override.
// Keys is a list of key strings (e.g. ["q", "ctrl+c"]).
type KeyBindingConfig struct {
	Keys []string `yaml:"keys,omitempty"`
}

// KeysConfig holds all key binding overrides.
// Any unset binding retains its compiled default.
type KeysConfig struct {
	Quit            *KeyBindingConfig `yaml:"quit,omitempty"`
	Help            *KeyBindingConfig `yaml:"help,omitempty"`
	Back            *KeyBindingConfig `yaml:"back,omitempty"`
	Enter           *KeyBindingConfig `yaml:"enter,omitempty"`
	Command         *KeyBindingConfig `yaml:"command,omitempty"`
	Filter          *KeyBindingConfig `yaml:"filter,omitempty"`
	Up              *KeyBindingConfig `yaml:"up,omitempty"`
	Down            *KeyBindingConfig `yaml:"down,omitempty"`
	PageUp          *KeyBindingConfig `yaml:"pageUp,omitempty"`
	PageDown        *KeyBindingConfig `yaml:"pageDown,omitempty"`
	Top             *KeyBindingConfig `yaml:"top,omitempty"`
	Bottom          *KeyBindingConfig `yaml:"bottom,omitempty"`
	CancelInput     *KeyBindingConfig `yaml:"cancelInput,omitempty"`
	Tab             *KeyBindingConfig `yaml:"tab,omitempty"`
	ScaleUp         *KeyBindingConfig `yaml:"scaleUp,omitempty"`
	ScaleDown       *KeyBindingConfig `yaml:"scaleDown,omitempty"`
	Deploy          *KeyBindingConfig `yaml:"deploy,omitempty"`
	Relate          *KeyBindingConfig `yaml:"relate,omitempty"`
	DeleteRelation  *KeyBindingConfig `yaml:"deleteRelation,omitempty"`
	LogsJump        *KeyBindingConfig `yaml:"logsJump,omitempty"`
	LogsView        *KeyBindingConfig `yaml:"logsView,omitempty"`
	ClearFilter     *KeyBindingConfig `yaml:"clearFilter,omitempty"`
	SearchOpen      *KeyBindingConfig `yaml:"searchOpen,omitempty"`
	SearchNext      *KeyBindingConfig `yaml:"searchNext,omitempty"`
	SearchPrev      *KeyBindingConfig `yaml:"searchPrev,omitempty"`
	FilterOpen      *KeyBindingConfig `yaml:"filterOpen,omitempty"`
	UnitsNav        *KeyBindingConfig `yaml:"unitsNav,omitempty"`
	ApplicationsNav *KeyBindingConfig `yaml:"applicationsNav,omitempty"`
	RelationsNav    *KeyBindingConfig `yaml:"relationsNav,omitempty"`
	SecretsNav      *KeyBindingConfig `yaml:"secretsNav,omitempty"`
	MachinesNav     *KeyBindingConfig `yaml:"machinesNav,omitempty"`
	OffersNav       *KeyBindingConfig `yaml:"offersNav,omitempty"`
	StorageNav      *KeyBindingConfig `yaml:"storageNav,omitempty"`
	Decode          *KeyBindingConfig `yaml:"decode,omitempty"`
	Yank            *KeyBindingConfig `yaml:"yank,omitempty"`
	ApplyFilter     *KeyBindingConfig `yaml:"applyFilter,omitempty"`
	Right           *KeyBindingConfig `yaml:"right,omitempty"`
	Left            *KeyBindingConfig `yaml:"left,omitempty"`
	RunAction       *KeyBindingConfig `yaml:"runAction,omitempty"`
	ConfigNav       *KeyBindingConfig `yaml:"configNav,omitempty"`
	ChatNav         *KeyBindingConfig `yaml:"chatNav,omitempty"`
}

// NewDefault returns a Config with all compiled defaults.
func NewDefault() *Config {
	return &Config{
		Jara: JaraConfig{
			RefreshRate:   DefaultRefreshRate,
			LogLevel:      DefaultLogLevel,
			CharmhubURL:   DefaultCharmhubURL,
			ToastDuration: DefaultToastDuration,
		},
	}
}

// Load reads configuration from a YAML file. If the file does not exist,
// the config is left unchanged (defaults apply). Other errors are returned.
func (c *Config) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("parsing config file %s: %w", path, err)
	}

	// Re-apply defaults for zero values that should have non-zero defaults.
	if c.Jara.RefreshRate == 0 {
		c.Jara.RefreshRate = DefaultRefreshRate
	}
	if c.Jara.LogLevel == "" {
		c.Jara.LogLevel = DefaultLogLevel
	}
	if c.Jara.CharmhubURL == "" {
		c.Jara.CharmhubURL = DefaultCharmhubURL
	}

	// Clamp RefreshRate to a sensible range.
	minRate := 0.5  // 500ms
	maxRate := 60.0 // 60s
	if c.Jara.RefreshRate < minRate {
		c.Jara.RefreshRate = minRate
	}
	if c.Jara.RefreshRate > maxRate {
		c.Jara.RefreshRate = maxRate
	}

	// Validate LogLevel.
	switch strings.ToLower(c.Jara.LogLevel) {
	case "error", "warn", "info", "debug", "trace":
		c.Jara.LogLevel = strings.ToLower(c.Jara.LogLevel)
	default:
		c.Jara.LogLevel = DefaultLogLevel
	}

	// Apply default for toast duration.
	if c.Jara.ToastDuration <= 0 {
		c.Jara.ToastDuration = DefaultToastDuration
	}

	return nil
}

// RefreshDuration returns the config's refresh rate as a time.Duration.
func (c *Config) RefreshDuration() time.Duration {
	rate := c.Jara.RefreshRate
	if rate <= 0 {
		rate = DefaultRefreshRate
	}
	return time.Duration(rate * float64(time.Second))
}

// Save writes the current configuration to a YAML file, creating
// parent directories as needed.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
