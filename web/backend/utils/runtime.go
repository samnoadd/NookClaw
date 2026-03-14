package utils

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/samnoadd/NookClaw/pkg/appenv"
)

// GetDefaultConfigPath returns the active NookClaw config path.
func GetDefaultConfigPath() string {
	path := appenv.ConfigPath()
	if path == "" {
		return "config.json"
	}
	return path
}

// FindNookclawBinary locates the NookClaw executable.
// Search order:
//  1. NOOKCLAW_BINARY environment variable (explicit override)
//  2. PICOCLAW_BINARY environment variable (legacy explicit override)
//  3. Same directory as the current executable
//  4. Falls back to "nookclaw" on $PATH, then the legacy "picoclaw"
func FindNookclawBinary() string {
	binaryNames := []string{appenv.BinaryName, appenv.LegacyBinaryName}
	if runtime.GOOS == "windows" {
		binaryNames = []string{appenv.BinaryName + ".exe", appenv.LegacyBinaryName + ".exe"}
	}

	for _, envName := range []string{appenv.BinaryEnv, appenv.LegacyBinaryEnv} {
		if p := os.Getenv(envName); p != "" {
			if info, _ := os.Stat(p); info != nil && !info.IsDir() {
				return p
			}
		}
	}

	if exe, err := os.Executable(); err == nil {
		for _, binaryName := range binaryNames {
			candidate := filepath.Join(filepath.Dir(exe), binaryName)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate
			}
		}
	}

	for _, binaryName := range binaryNames {
		if _, err := exec.LookPath(binaryName); err == nil {
			return binaryName
		}
	}

	return binaryNames[0]
}

// GetLocalIP returns the local IP address of the machine.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return ""
}

// OpenBrowser automatically opens the given URL in the default browser.
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
