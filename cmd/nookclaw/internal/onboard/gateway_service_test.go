package onboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderGatewayUserService(t *testing.T) {
	unit := renderGatewayUserService("/usr/local/bin/nookclaw", "/tmp/nookclaw/config.json")

	for _, snippet := range []string{
		"Description=NookClaw Gateway",
		`ExecStart="/usr/local/bin/nookclaw" gateway`,
		`Environment=NOOKCLAW_CONFIG="/tmp/nookclaw/config.json"`,
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, snippet) {
			t.Fatalf("expected unit file to contain %q\nunit:\n%s", snippet, unit)
		}
	}
}

func TestInstallGatewayUserService_WritesAndEnablesUnit(t *testing.T) {
	originalLookPath := gatewayLookPath
	originalSupports := gatewaySupportsUserService
	originalRun := gatewayRun
	originalExecutablePath := gatewayExecutablePath
	originalUserConfigDir := gatewayUserConfigDir
	originalCurrentUser := gatewayCurrentUser
	defer func() {
		gatewayLookPath = originalLookPath
		gatewaySupportsUserService = originalSupports
		gatewayRun = originalRun
		gatewayExecutablePath = originalExecutablePath
		gatewayUserConfigDir = originalUserConfigDir
		gatewayCurrentUser = originalCurrentUser
	}()

	tempDir := t.TempDir()
	var commands [][]string
	gatewayLookPath = func(file string) (string, error) { return "/usr/bin/systemctl", nil }
	gatewaySupportsUserService = func() bool { return true }
	gatewayRun = func(name string, args ...string) error {
		commands = append(commands, append([]string{name}, args...))
		return nil
	}
	gatewayExecutablePath = func() (string, error) { return "/usr/local/bin/nookclaw", nil }
	gatewayUserConfigDir = func() (string, error) { return tempDir, nil }
	gatewayCurrentUser = func() (string, error) { return "shayea", nil }

	result := installGatewayUserService("/tmp/nookclaw/config.json")
	if !result.Enabled {
		t.Fatalf("expected service to be enabled, got %+v", result)
	}
	unitPath := filepath.Join(tempDir, "systemd", "user", gatewayUserServiceName)
	if result.UnitPath != unitPath {
		t.Fatalf("UnitPath = %q, want %q", result.UnitPath, unitPath)
	}
	if _, err := os.Stat(unitPath); err != nil {
		t.Fatalf("expected unit file to exist: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("expected two systemctl commands, got %v", commands)
	}
	if got := strings.Join(commands[0], " "); got != "systemctl --user daemon-reload" {
		t.Fatalf("first command = %q", got)
	}
	if got := strings.Join(commands[1], " "); got != "systemctl --user enable --now nookclaw-gateway.service" {
		t.Fatalf("second command = %q", got)
	}
}
