package services

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/bambuo/rclick-go/internal/config"
	"github.com/bambuo/rclick-go/internal/ipc"
)

var ErrHelperUnavailable = errors.New("native helper unavailable")

// FileOperations performs the file actions on behalf of the Finder menu.
// Destructive and native operations go through the helper so the process
// behaves like the original sandboxed app.
type FileOperations struct {
	Store     *config.Store
	Helper    *ipc.Helper
	Clipboard func(text string) error
	Home      string
}

func (f *FileOperations) home() string {
	if f.Home != "" {
		return f.Home
	}
	return config.HomeDir()
}

func (f *FileOperations) CopyPaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	text := Decode(paths[0])
	if f.Clipboard == nil {
		return fmt.Errorf("clipboard unavailable")
	}
	return f.Clipboard(text)
}

func (f *FileOperations) Delete(paths []string, trigger string) error {
	if trigger == "ctx-container" {
		log.Printf("delete: refusing to delete the container itself")
		return nil
	}
	home := f.home()
	var targets []string
	for _, path := range paths {
		decoded := Decode(path)
		if IsProtected(decoded, home) {
			log.Printf("delete: skipped protected path %s", decoded)
			continue
		}
		targets = append(targets, decoded)
	}
	if len(targets) == 0 {
		return nil
	}
	if f.Helper == nil {
		return ErrHelperUnavailable
	}
	return f.Helper.Trash(targets)
}

func (f *FileOperations) SetHidden(paths []string, hidden bool, trigger string) error {
	home := f.home()
	var targets []string

	if trigger == "ctx-container" && len(paths) > 0 {
		container := Decode(paths[0])
		targets = append(targets, children(container, home)...)
		if !hidden {
			targets = append(targets, container)
		}
	} else {
		for _, path := range paths {
			targets = append(targets, Decode(path))
		}
	}

	seen := make(map[string]struct{}, len(targets))
	safe := targets[:0]
	for _, path := range targets {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		if IsProtected(path, home) {
			continue
		}
		if _, err := os.Lstat(path); err != nil {
			continue
		}
		safe = append(safe, path)
	}
	if len(safe) == 0 {
		return nil
	}
	if f.Helper == nil {
		return ErrHelperUnavailable
	}
	return f.Helper.SetHidden(safe, hidden)
}

func (f *FileOperations) AirDrop(paths []string, trigger string) error {
	if trigger == "ctx-container" {
		return nil
	}
	home := f.home()
	var files []string
	for _, path := range paths {
		decoded := Decode(path)
		if IsProtected(decoded, home) {
			continue
		}
		info, err := os.Stat(decoded)
		if err != nil || info.IsDir() {
			continue
		}
		files = append(files, decoded)
	}
	if len(files) == 0 {
		return nil
	}
	if f.Helper == nil {
		return ErrHelperUnavailable
	}
	return f.Helper.AirDrop(files)
}

func (f *FileOperations) Open(paths []string) error {
	var decoded []string
	for _, path := range paths {
		if value := Decode(path); value != "" {
			decoded = append(decoded, value)
		}
	}
	if len(decoded) == 0 {
		return nil
	}
	if f.Helper == nil {
		return ErrHelperUnavailable
	}
	return f.Helper.Open(decoded)
}

func (f *FileOperations) CreateFile(directory, newFileID string) error {
	cfg := f.Store.Current()
	var fileType *config.NewFile
	for i := range cfg.NewFiles {
		if cfg.NewFiles[i].ID == newFileID && cfg.NewFiles[i].Enabled {
			fileType = &cfg.NewFiles[i]
			break
		}
	}
	if fileType == nil {
		return fmt.Errorf("new file type %q not found", newFileID)
	}

	dir := Decode(directory)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("create file: %s is not a directory", dir)
	}

	target := PlanNewFilePath(dir, fileType.Ext, cfg.Defaults.NewFileName, nil)
	if fileType.Template != "" {
		if info, err := os.Stat(fileType.Template); err == nil && !info.IsDir() {
			if err := copyFileExclusive(fileType.Template, target); err != nil {
				return fmt.Errorf("create file from template: %w", err)
			}
			log.Printf("created %s from %s", target, fileType.Template)
			return nil
		}
	}

	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	log.Printf("created %s", target)
	return nil
}

func children(directory, home string) []string {
	if IsProtected(directory, home) {
		return nil
	}
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, filepath.Join(directory, entry.Name()))
	}
	return out
}

func copyFileExclusive(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
