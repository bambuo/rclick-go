package services

import "strings"

var systemPrefixes = []string{
	"/Applications",
	"/System",
	"/Library",
	"/usr",
	"/bin",
	"/sbin",
	"/var",
}

// ForHome lists the user-owned paths the app refuses to delete or hide.
func ForHome(home string) []string {
	home = strings.TrimRight(home, "/")
	return []string{
		home,
		home + "/Desktop",
		home + "/Desktop/danger",
		home + "/Applications",
	}
}

func IsProtected(path, home string) bool {
	normalized := normalizePath(path)
	for _, protected := range ForHome(home) {
		if normalized == normalizePath(protected) {
			return true
		}
	}
	for _, prefix := range systemPrefixes {
		if normalized == prefix || strings.HasPrefix(normalized, prefix+"/") {
			return true
		}
	}
	return normalized == "/Users"
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if len(trimmed) > 1 {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed
}
