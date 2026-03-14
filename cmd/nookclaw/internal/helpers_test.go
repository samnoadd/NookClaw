package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigPath(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/tmp/home", ".nookclaw", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithNOOKCLAW_HOME(t *testing.T) {
	t.Setenv("NOOKCLAW_HOME", "/custom/picoclaw")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/custom/picoclaw", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithNOOKCLAW_CONFIG(t *testing.T) {
	t.Setenv("NOOKCLAW_CONFIG", "/custom/config.json")
	t.Setenv("NOOKCLAW_HOME", "/custom/picoclaw")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := "/custom/config.json"

	assert.Equal(t, want, got)
}

func TestGetConfigPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific HOME behavior varies; run on windows")
	}

	testUserProfilePath := `C:\Users\Test`
	t.Setenv("USERPROFILE", testUserProfilePath)

	got := GetConfigPath()
	want := filepath.Join(testUserProfilePath, ".nookclaw", "config.json")

	require.True(t, strings.EqualFold(got, want), "GetConfigPath() = %q, want %q", got, want)
}

func TestGetDefaultConfigPath_IgnoresLegacyConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	legacyConfig := filepath.Join(home, ".picoclaw", "config.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyConfig), 0o755))
	require.NoError(t, os.WriteFile(legacyConfig, []byte("{}"), 0o644))

	got := GetDefaultConfigPath()
	want := filepath.Join(home, ".nookclaw", "config.json")

	assert.Equal(t, want, got)
}
