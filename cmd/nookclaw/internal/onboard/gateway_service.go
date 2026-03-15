package onboard

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
)

const gatewayUserServiceName = "nookclaw-gateway.service"

type gatewayServiceSetup struct {
	UnitPath   string
	Enabled    bool
	Error      string
	LingerHint string
}

var (
	gatewayLookPath = exec.LookPath
	gatewaySupportsUserService = func() bool {
		if runtime.GOOS != "linux" {
			return false
		}
		_, err := gatewayLookPath("systemctl")
		return err == nil
	}
	gatewayRun = func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		return cmd.Run()
	}
	gatewayExecutablePath = os.Executable
	gatewayUserConfigDir  = os.UserConfigDir
	gatewayCurrentUser    = func() (string, error) {
		current, err := user.Current()
		if err != nil {
			return "", err
		}
		return current.Username, nil
	}
)

func gatewayUserServiceSupported() bool {
	return gatewaySupportsUserService()
}

func installGatewayUserService(configPath string) gatewayServiceSetup {
	result := gatewayServiceSetup{}

	executablePath, err := gatewayExecutablePath()
	if err != nil {
		result.Error = fmt.Sprintf("could not resolve the current nookclaw binary: %v", err)
		return result
	}

	userConfigDir, err := gatewayUserConfigDir()
	if err != nil {
		result.Error = fmt.Sprintf("could not resolve the user config directory: %v", err)
		return result
	}

	unitDir := filepath.Join(userConfigDir, "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		result.Error = fmt.Sprintf("could not create the user service directory: %v", err)
		return result
	}

	unitPath := filepath.Join(unitDir, gatewayUserServiceName)
	if err := os.WriteFile(unitPath, []byte(renderGatewayUserService(executablePath, configPath)), 0o644); err != nil {
		result.Error = fmt.Sprintf("could not write the gateway service file: %v", err)
		return result
	}
	result.UnitPath = unitPath
	result.LingerHint = gatewayLingerHint()

	if !gatewayUserServiceSupported() {
		result.Error = "service file was written, but systemctl --user is not available on this machine"
		return result
	}

	if err := gatewayRun("systemctl", "--user", "daemon-reload"); err != nil {
		result.Error = fmt.Sprintf("service file was written, but `systemctl --user daemon-reload` failed: %v", err)
		return result
	}
	if err := gatewayRun("systemctl", "--user", "enable", "--now", gatewayUserServiceName); err != nil {
		result.Error = fmt.Sprintf("service file was written, but `systemctl --user enable --now %s` failed: %v", gatewayUserServiceName, err)
		return result
	}

	result.Enabled = true
	return result
}

func renderGatewayUserService(executablePath, configPath string) string {
	return fmt.Sprintf(`[Unit]
Description=NookClaw Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%q gateway
WorkingDirectory=%%h
Environment=NOOKCLAW_CONFIG=%q
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
`, executablePath, configPath)
}

func gatewayLingerHint() string {
	username, err := gatewayCurrentUser()
	if err != nil || username == "" {
		return "To start before login, run: sudo loginctl enable-linger $USER"
	}
	return fmt.Sprintf("To start before login, run: sudo loginctl enable-linger %s", username)
}
