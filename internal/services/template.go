package services

import (
	"path/filepath"
	"strings"
)

// ResolveTemplate substitutes the placeholders documented by RClick. It
// mirrors QuickCommand.resolvedCommand(forPaths:) in the Swift app: the
// first path fills the singular placeholders, and {dir} is the directory
// plus the file name without its extension.
func ResolveTemplate(template string, paths []string) string {
	quoted := make([]string, len(paths))
	for i, path := range paths {
		quoted[i] = `"` + path + `"`
	}
	resolved := strings.ReplaceAll(template, "{all}", strings.Join(quoted, " "))
	if len(paths) == 0 {
		return resolved
	}

	first := paths[0]
	base := filepath.Base(first)
	ext := filepath.Ext(first)
	withoutExt := strings.TrimSuffix(base, ext)
	dir := filepath.Join(filepath.Dir(first), withoutExt)

	replacements := []struct{ old, new string }{
		{"{path}", first},
		{"{name}", base},
		{"{dir}", dir},
		{"{ext}", ext},
		{"{name_no_ext}", withoutExt},
	}
	for _, replacement := range replacements {
		resolved = strings.ReplaceAll(resolved, replacement.old, replacement.new)
	}
	return resolved
}
