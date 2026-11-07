package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// HomeDir returns the real home directory, following the user record
// rather than $HOME so sandboxed and helper processes agree.
func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join("/Users", os.Getenv("USER"))
	}
	return home
}

func RootDir() string {
	return filepath.Join(HomeDir(), "Library", "Application Support", "RClick")
}

func ConfigFile() string {
	return filepath.Join(RootDir(), "config.json")
}

func LogDir() string {
	return filepath.Join(HomeDir(), "Library", "Logs", "RClick")
}

// EnsureRoot creates the configuration directory with user-only access,
// as the original app's sandbox container does.
func EnsureRoot() error {
	root := RootDir()
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return fmt.Errorf("chmod config directory: %w", err)
	}
	return nil
}
