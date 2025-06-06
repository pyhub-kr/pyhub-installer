package config

import (
	"fmt"
	"runtime"
)

// Config holds application configuration
type Config struct {
	// Download settings
	ChunkSize   int64 `json:"chunk_size"`
	Parallelism int   `json:"parallelism"`
	Timeout     int   `json:"timeout_seconds"`

	// Installation settings
	DefaultInstallPath string `json:"default_install_path"`
	DefaultChmod       string `json:"default_chmod"`

	// Verification settings
	VerifyByDefault bool `json:"verify_by_default"`
	ExtractByDefault bool `json:"extract_by_default"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	config := &Config{
		ChunkSize:        1024 * 1024, // 1MB chunks
		Parallelism:      4,           // 4 parallel downloads
		Timeout:          300,         // 5 minutes
		VerifyByDefault:  true,
		ExtractByDefault: true,
		DefaultChmod:     "755",
	}

	// Platform-specific defaults
	switch runtime.GOOS {
	case "windows":
		config.DefaultInstallPath = "C:\\Program Files\\pyhub-installer"
	case "darwin":
		config.DefaultInstallPath = "/usr/local/bin"
	case "linux":
		config.DefaultInstallPath = "/usr/local/bin"
	default:
		config.DefaultInstallPath = "./bin"
	}

	return config
}

// Validate validates configuration
func (c *Config) Validate() error {
	if c.ChunkSize <= 0 {
		return fmt.Errorf("chunk_size must be positive")
	}
	if c.Parallelism <= 0 {
		return fmt.Errorf("parallelism must be positive")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.DefaultInstallPath == "" {
		return fmt.Errorf("default_install_path cannot be empty")
	}
	return nil
}