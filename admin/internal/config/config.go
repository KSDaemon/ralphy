// Package config handles loading ralphy-admin settings from
// ~/.config/ralphy/settings.toml with sensible defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds all ralphy-admin configuration values.
type Config struct {
	// StaleThreshold is how long since the last heartbeat before a running
	// session is considered "stale". Default: 5m.
	StaleThreshold time.Duration

	// SessionTTL is how long to keep session files before auto-cleanup.
	// Default: 24h.
	SessionTTL time.Duration

	// DefaultTool is the AI tool ralphy launches when --tool is not passed
	// on the command line. One of "opencode", "amp", "claude". Default: "opencode".
	// Overridden by the RALPHY_TOOL environment variable.
	DefaultTool string

	// DefaultModel is the model name passed verbatim to the AI tool when
	// --model is not provided. Format depends on the tool (claude accepts
	// aliases like "sonnet"; opencode wants "provider/model"; amp ignores it).
	// Empty = let the tool pick its own default.
	// Overridden by the RALPHY_MODEL environment variable.
	DefaultModel string
}

// tomlConfig mirrors the TOML file structure with string durations.
type tomlConfig struct {
	StaleThreshold string `toml:"stale_threshold"`
	SessionTTL     string `toml:"session_ttl"`
	DefaultTool    string `toml:"default_tool"`
	DefaultModel   string `toml:"default_model"`
}

// validTools is the set of AI tools ralphy supports.
var validTools = map[string]struct{}{
	"opencode": {},
	"amp":      {},
	"claude":   {},
}

func isValidTool(s string) bool {
	_, ok := validTools[s]
	return ok
}

// DefaultConfig returns a Config with built-in defaults.
func DefaultConfig() *Config {
	return &Config{
		StaleThreshold: 5 * time.Minute,
		SessionTTL:     24 * time.Hour,
		DefaultTool:    "opencode",
		DefaultModel:   "",
	}
}

// Load reads configuration from ~/.config/ralphy/settings.toml.
// Missing file or missing keys silently fall back to defaults.
// The RALPHY_TOOL and RALPHY_MODEL environment variables, when set,
// override default_tool and default_model from the config file respectively.
// Returns an error only if the file exists but is malformed, or if a
// configured AI tool name is not one of the supported tools.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	configPath := configFilePath()
	data, err := os.ReadFile(configPath)
	if err == nil {
		var tc tomlConfig
		if err := toml.Unmarshal(data, &tc); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", configPath, err)
		}

		if tc.StaleThreshold != "" {
			d, err := parseDuration(tc.StaleThreshold)
			if err != nil {
				return nil, fmt.Errorf("invalid stale_threshold %q in %s: %w", tc.StaleThreshold, configPath, err)
			}
			cfg.StaleThreshold = d
		}

		if tc.SessionTTL != "" {
			d, err := parseDuration(tc.SessionTTL)
			if err != nil {
				return nil, fmt.Errorf("invalid session_ttl %q in %s: %w", tc.SessionTTL, configPath, err)
			}
			cfg.SessionTTL = d
		}

		if tc.DefaultTool != "" {
			if !isValidTool(tc.DefaultTool) {
				return nil, fmt.Errorf("invalid default_tool %q in %s: must be one of opencode, amp, claude", tc.DefaultTool, configPath)
			}
			cfg.DefaultTool = tc.DefaultTool
		}

		if tc.DefaultModel != "" {
			cfg.DefaultModel = tc.DefaultModel
		}
	}

	if envTool := strings.TrimSpace(os.Getenv("RALPHY_TOOL")); envTool != "" {
		if !isValidTool(envTool) {
			return nil, fmt.Errorf("invalid RALPHY_TOOL %q: must be one of opencode, amp, claude", envTool)
		}
		cfg.DefaultTool = envTool
	}

	if envModel := strings.TrimSpace(os.Getenv("RALPHY_MODEL")); envModel != "" {
		cfg.DefaultModel = envModel
	}

	return cfg, nil
}

// configFilePath returns ~/.config/ralphy/settings.toml.
func configFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".config", "ralphy", "settings.toml")
}

// parseDuration parses a human-friendly duration string.
// Supported formats:
//
//	30s, 5m, 2h, 1d, 7d
//	1h30m, 2d12h
//	plain Go durations like "5m0s" also work
//
// Units: s (seconds), m (minutes), h (hours), d (days).
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// If it contains 'd' (days), we need custom parsing since Go's
	// time.ParseDuration doesn't support days.
	if strings.Contains(s, "d") {
		return parseDurationWithDays(s)
	}

	// Try standard Go duration parsing (handles "5m", "1h30m", "30s", etc.)
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: use formats like 30s, 5m, 2h, 1d, 7d", s)
	}
	return d, nil
}

// durationPartRe matches a single component like "7d", "2h", "30m", "10s".
var durationPartRe = regexp.MustCompile(`(\d+)\s*([dhms])`)

func parseDurationWithDays(s string) (time.Duration, error) {
	parts := durationPartRe.FindAllStringSubmatch(s, -1)
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid duration %q: use formats like 30s, 5m, 2h, 1d, 7d", s)
	}

	var total time.Duration
	for _, part := range parts {
		n, err := strconv.Atoi(part[1])
		if err != nil {
			return 0, fmt.Errorf("invalid number in duration %q", s)
		}
		switch part[2] {
		case "d":
			total += time.Duration(n) * 24 * time.Hour
		case "h":
			total += time.Duration(n) * time.Hour
		case "m":
			total += time.Duration(n) * time.Minute
		case "s":
			total += time.Duration(n) * time.Second
		}
	}

	if total == 0 {
		return 0, fmt.Errorf("duration %q resolves to zero", s)
	}

	return total, nil
}
