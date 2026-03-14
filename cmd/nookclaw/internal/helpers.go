package internal

import (
	"github.com/samnoadd/NookClaw/pkg/appenv"
	"github.com/samnoadd/NookClaw/pkg/config"
)

const Logo = "N>"

// GetNookclawHome returns the active NookClaw home directory.
func GetNookclawHome() string {
	return appenv.ResolveHomeDir()
}

func GetConfigPath() string {
	return appenv.ConfigPath()
}

func GetDefaultConfigPath() string {
	return appenv.DefaultConfigPath()
}

func LoadConfig() (*config.Config, error) {
	return config.LoadConfig(GetConfigPath())
}

// FormatVersion returns the version string with optional git commit
// Deprecated: Use pkg/config.FormatVersion instead
func FormatVersion() string {
	return config.FormatVersion()
}

// FormatBuildInfo returns build time and go version info
// Deprecated: Use pkg/config.FormatBuildInfo instead
func FormatBuildInfo() (string, string) {
	return config.FormatBuildInfo()
}

// GetVersion returns the version string
// Deprecated: Use pkg/config.GetVersion instead
func GetVersion() string {
	return config.GetVersion()
}
