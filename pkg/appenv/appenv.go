package appenv

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	BinaryName        = "nookclaw"
	LegacyBinaryName  = "picoclaw"
	HomeEnv           = "NOOKCLAW_HOME"
	LegacyHomeEnv     = "PICOCLAW_HOME"
	ConfigEnv         = "NOOKCLAW_CONFIG"
	LegacyConfigEnv   = "PICOCLAW_CONFIG"
	BuiltinSkillsEnv  = "NOOKCLAW_BUILTIN_SKILLS"
	LegacySkillsEnv   = "PICOCLAW_BUILTIN_SKILLS"
	GatewayHostEnv    = "NOOKCLAW_GATEWAY_HOST"
	LegacyGatewayEnv  = "PICOCLAW_GATEWAY_HOST"
	BinaryEnv         = "NOOKCLAW_BINARY"
	LegacyBinaryEnv   = "PICOCLAW_BINARY"
	DefaultDirName    = ".nookclaw"
	LegacyDefaultDir  = ".picoclaw"
	defaultConfigFile = "config.json"
	defaultAuthFile   = "auth.json"
	defaultWorkspace  = "workspace"
	defaultSkillsDir  = "skills"
)

// Getenv returns the preferred env var value, falling back to the legacy name.
func Getenv(primary, legacy string) string {
	if value := strings.TrimSpace(os.Getenv(primary)); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(legacy))
}

// ApplyCompatibility mirrors NOOKCLAW_* env vars into their legacy PICOCLAW_*
// equivalents so existing config tags and subprocesses continue to work.
func ApplyCompatibility() error {
	for _, raw := range os.Environ() {
		key, value, ok := strings.Cut(raw, "=")
		if !ok || !strings.HasPrefix(key, "NOOKCLAW_") {
			continue
		}
		legacyKey := "PICOCLAW_" + strings.TrimPrefix(key, "NOOKCLAW_")
		if strings.TrimSpace(os.Getenv(legacyKey)) != "" {
			continue
		}
		if err := os.Setenv(legacyKey, value); err != nil {
			return err
		}
	}
	return nil
}

func homeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

func pathExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info != nil
}

func userPath(dirName string) string {
	home := homeDir()
	if home == "" {
		return dirName
	}
	return filepath.Join(home, dirName)
}

func newHomeDir() string {
	return userPath(DefaultDirName)
}

func legacyHomeDir() string {
	return userPath(LegacyDefaultDir)
}

// DefaultHomeDir returns the publish-facing default home for fresh installs.
func DefaultHomeDir() string {
	if home := Getenv(HomeEnv, LegacyHomeEnv); home != "" {
		return home
	}
	return newHomeDir()
}

// ResolveHomeDir returns the most compatible home location for runtime access.
// It prefers explicit overrides, then an existing NookClaw home, then an
// existing legacy PicoClaw home, and finally the new default.
func ResolveHomeDir() string {
	if home := Getenv(HomeEnv, LegacyHomeEnv); home != "" {
		return home
	}
	if pathExists(newHomeDir()) {
		return newHomeDir()
	}
	if pathExists(legacyHomeDir()) {
		return legacyHomeDir()
	}
	return newHomeDir()
}

func preferredPath(filename string) string {
	if home := Getenv(HomeEnv, LegacyHomeEnv); home != "" {
		return filepath.Join(home, filename)
	}
	newPath := filepath.Join(newHomeDir(), filename)
	if pathExists(newPath) {
		return newPath
	}
	legacyPath := filepath.Join(legacyHomeDir(), filename)
	if pathExists(legacyPath) {
		return legacyPath
	}
	return newPath
}

// ConfigPath resolves the config file path, preferring new locations but
// falling back to an existing legacy config file when present.
func ConfigPath() string {
	if path := Getenv(ConfigEnv, LegacyConfigEnv); path != "" {
		return path
	}
	return preferredPath(defaultConfigFile)
}

// DefaultConfigPath returns the config path that new installs should create.
func DefaultConfigPath() string {
	if path := Getenv(ConfigEnv, LegacyConfigEnv); path != "" {
		return path
	}
	return filepath.Join(DefaultHomeDir(), defaultConfigFile)
}

// DefaultWorkspacePath returns the workspace path that new installs should use.
func DefaultWorkspacePath() string {
	return filepath.Join(DefaultHomeDir(), defaultWorkspace)
}

// ResolveWorkspacePath returns the active workspace path, with compatibility
// fallback to legacy data when no new workspace exists yet.
func ResolveWorkspacePath() string {
	return preferredPath(defaultWorkspace)
}

// AuthPath resolves the auth store location with legacy fallback.
func AuthPath() string {
	return preferredPath(defaultAuthFile)
}

// GlobalSkillsPath resolves the global skills directory with legacy fallback.
func GlobalSkillsPath() string {
	return preferredPath(defaultSkillsDir)
}

// BuiltinSkillsPath returns the optional builtin skills override path.
func BuiltinSkillsPath() string {
	return Getenv(BuiltinSkillsEnv, LegacySkillsEnv)
}
