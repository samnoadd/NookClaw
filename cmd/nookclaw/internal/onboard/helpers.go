package onboard

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal"
	"github.com/samnoadd/NookClaw/pkg/config"
)

func onboard() {
	configPath := internal.GetDefaultConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		fmt.Print("Overwrite? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" {
			fmt.Println("Aborted.")
			return
		}
	}

	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)

	fmt.Printf("%s NookClaw is ready!\n", internal.Logo)
	fmt.Println("\nThis personal fork starts in local-first mode:")
	fmt.Println("  - default model alias: private-local (Ollama)")
	fmt.Println("  - heartbeat disabled")
	fmt.Println("  - web search, web fetch, and remote skill registry disabled")
	fmt.Println("  - remote command targets disabled")
	fmt.Println("  - config and workspace default to ~/.nookclaw")
	fmt.Println("  - existing ~/.picoclaw installs are still detected automatically")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Confirm the default Ollama model alias in", configPath)
	fmt.Println("")
	fmt.Println("     The fork uses model_name \"private-local\" by default.")
	fmt.Println("     It points to ollama/qwen3.5:latest. Change it if your local model differs.")
	fmt.Println("")
	fmt.Println("  2. Chat locally: nookclaw agent -m \"Hello!\"")
	fmt.Println("")
	fmt.Println("  3. Enable any remote channel or provider only if you want it.")
}

func createWorkspaceTemplates(workspace string) {
	err := copyEmbeddedToTarget(workspace)
	if err != nil {
		fmt.Printf("Error copying workspace templates: %v\n", err)
	}
}

func copyEmbeddedToTarget(targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return fmt.Errorf("Failed to create target directory: %w", err)
	}

	// Walk through all files in embed.FS
	err := fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("Failed to read embedded file %s: %w", path, err)
		}

		new_path, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("Failed to get relative path for %s: %v\n", path, err)
		}

		// Build target file path
		targetPath := filepath.Join(targetDir, new_path)

		// Ensure target file's directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			return fmt.Errorf("Failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0o600); err != nil {
			return fmt.Errorf("Failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	return err
}
