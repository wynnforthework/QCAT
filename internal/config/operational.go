package config

// OperationalConfig captures run-mode specific operational requirements.
type OperationalConfig struct {
	Mode        string                           `yaml:"mode"`
	Modes       map[string]OperationalModeConfig `yaml:"modes"`
	KeyProfiles map[string]OperationalKeyProfile `yaml:"key_profiles"`
}

// OperationalModeConfig defines expectations for a specific runtime mode.
type OperationalModeConfig struct {
	Description         string   `yaml:"description"`
	RequiredKeyProfile  string   `yaml:"required_key_profile"`
	OptionalKeyProfiles []string `yaml:"optional_key_profiles"`
	RequireDoubleLoop   bool     `yaml:"require_double_loop"`
	RequireTrading      bool     `yaml:"require_trading"`
	RequireStrategy     bool     `yaml:"require_strategy"`
}

// OperationalKeyProfile describes a concrete exchange credential profile.
type OperationalKeyProfile struct {
	Exchange            string   `yaml:"exchange"`
	Market              string   `yaml:"market"`
	Environment         string   `yaml:"environment"`
	BaseURL             string   `yaml:"base_url"`
	FuturesBaseURL      string   `yaml:"futures_base_url"`
	WebsocketURL        string   `yaml:"websocket_url"`
	FuturesWebsocketURL string   `yaml:"futures_websocket_url"`
	Permissions         []string `yaml:"permissions"`
	Notes               string   `yaml:"notes"`
}

// GetActiveMode returns the operational mode configuration for the active mode.
func (c *OperationalConfig) GetActiveMode() (string, *OperationalModeConfig) {
	if c == nil {
		return "", nil
	}
	mode := c.Mode
	if mode == "" {
		mode = "default"
	}
	if c.Modes == nil {
		return mode, nil
	}
	def, ok := c.Modes[mode]
	if !ok {
		return mode, nil
	}
	return mode, &def
}

// GetKeyProfile retrieves a configured key profile by name.
func (c *OperationalConfig) GetKeyProfile(name string) *OperationalKeyProfile {
	if c == nil || c.KeyProfiles == nil {
		return nil
	}
	profile, ok := c.KeyProfiles[name]
	if !ok {
		return nil
	}
	p := profile
	return &p
}
