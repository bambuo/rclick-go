package services

import (
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultBaseName = "Untitled"

// PlanNewFilePath returns the first free "<base><n><ext>" path in dir.
func PlanNewFilePath(dir, extension, baseName string, exists func(string) bool) string {
	if exists == nil {
		exists = fileExists
	}
	ext := NormalizeExtension(extension)
	name := strings.TrimSpace(baseName)
	if name == "" {
		name = DefaultBaseName
	}

	candidate := filepath.Join(dir, name+ext)
	counter := 1
	for exists(candidate) {
		candidate = filepath.Join(dir, name+strconv.Itoa(counter)+ext)
		counter++
	}
	return candidate
}

func NormalizeExtension(extension string) string {
	if extension == "" {
		return ""
	}
	if strings.HasPrefix(extension, ".") {
		return extension
	}
	return "." + extension
}
