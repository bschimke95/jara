package config

// Flags holds CLI flag overrides that take precedence over the config file.
// Pointer fields allow distinguishing "not set" (nil) from "set to zero value".
type Flags struct {
	RefreshRate *float64
	LogLevel    *string
	LogFile     *string
	Headless    *bool
	Logoless    *bool
	ReadOnly    *bool
	Command     *string
	ConfigFile  *string
	Demo        *bool
}

// NewFlags returns a Flags struct with all nil pointers (nothing overridden).
func NewFlags() *Flags {
	return &Flags{}
}

// Override applies non-nil flag values on top of the loaded Config.
// CLI flags always win over the config file.
func (c *Config) Override(flags *Flags) {
	if flags == nil {
		return
	}
	if flags.RefreshRate != nil && *flags.RefreshRate > 0 {
		c.Jara.RefreshRate = *flags.RefreshRate
	}
	if flags.LogLevel != nil && *flags.LogLevel != "" {
		c.Jara.LogLevel = *flags.LogLevel
	}
	if flags.LogFile != nil && *flags.LogFile != "" {
		c.Jara.LogFile = *flags.LogFile
	}
	if flags.Headless != nil {
		c.Jara.Headless = *flags.Headless
	}
	if flags.Logoless != nil {
		c.Jara.Logoless = *flags.Logoless
	}
	if flags.ReadOnly != nil {
		c.Jara.ReadOnly = *flags.ReadOnly
	}
}
