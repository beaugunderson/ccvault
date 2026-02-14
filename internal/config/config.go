// ABOUTME: Configuration management for ccvault
// ABOUTME: Handles config file loading, defaults, and environment overrides

package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all ccvault configuration
type Config struct {
	ClaudeHome string `mapstructure:"claude_home"`
	DataDir    string `mapstructure:"data_dir"`
}

// DefaultClaudeHome returns the default Claude Code data directory
func DefaultClaudeHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// DefaultDataDir returns the default ccvault data directory
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ccvault")
}

// Load reads configuration from file and environment
func Load() (*Config, error) {
	// Set defaults
	viper.SetDefault("claude_home", DefaultClaudeHome())
	viper.SetDefault("data_dir", DefaultDataDir())

	// Environment variables
	viper.SetEnvPrefix("CCVAULT")
	viper.AutomaticEnv()

	// Config file
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(DefaultDataDir())
	viper.AddConfigPath(".")

	// Read config file if it exists (not an error if missing)
	_ = viper.ReadInConfig()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// EnsureDataDir creates the data directory if it doesn't exist
func EnsureDataDir(cfg *Config) error {
	return os.MkdirAll(cfg.DataDir, 0755)
}
