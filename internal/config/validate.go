package config

import (
	"fmt"
	"sort"
	"strings"
)

type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "invalid RClick config: " + strings.Join(e.Problems, "; ")
}

func Validate(cfg *Config) error {
	var problems []string
	if cfg.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Sprintf("unsupported schemaVersion %d", cfg.SchemaVersion))
	}
	if cfg.Revision < 0 {
		problems = append(problems, "revision must be non-negative")
	}

	validateIDs(appIDs(cfg.Apps), "apps", &problems)
	validateIDs(actionIDs(cfg.Actions), "actions", &problems)
	validateIDs(newFileIDs(cfg.NewFiles), "newFiles", &problems)
	validateIDs(commonDirIDs(cfg.CommonDirs), "commonDirs", &problems)
	validateIDs(permissionDirIDs(cfg.PermissionDirs), "permissionDirs", &problems)
	validateIDs(quickCommandIDs(cfg.QuickCommands), "quickCommands", &problems)

	for _, app := range cfg.Apps {
		if strings.TrimSpace(app.URL) == "" {
			problems = append(problems, fmt.Sprintf("app %q has an empty url", app.ID))
		}
	}
	for _, command := range cfg.QuickCommands {
		if strings.TrimSpace(command.Template) == "" {
			problems = append(problems, fmt.Sprintf("quick command %q has an empty template", command.ID))
		}
	}

	if len(problems) > 0 {
		return &ValidationError{Problems: problems}
	}
	return nil
}

// Normalize keeps indices dense and in order, as the Finder extension
// builds menus in the order it reads them.
func Normalize(cfg *Config) {
	sort.SliceStable(cfg.Actions, func(i, j int) bool { return cfg.Actions[i].Idx < cfg.Actions[j].Idx })
	for i := range cfg.Actions {
		cfg.Actions[i].Idx = i
	}
	sort.SliceStable(cfg.NewFiles, func(i, j int) bool { return cfg.NewFiles[i].Idx < cfg.NewFiles[j].Idx })
	for i := range cfg.NewFiles {
		cfg.NewFiles[i].Idx = i
	}
	sort.SliceStable(cfg.QuickCommands, func(i, j int) bool { return cfg.QuickCommands[i].Idx < cfg.QuickCommands[j].Idx })
	for i := range cfg.QuickCommands {
		cfg.QuickCommands[i].Idx = i
	}
}

// NormalizeSlices replaces nil collections with empty ones so they encode
// as [] rather than null; the Swift decoder accepts both, but explicit
// arrays keep the file readable.
func NormalizeSlices(cfg *Config) {
	if cfg.Apps == nil {
		cfg.Apps = []OpenWithApp{}
	}
	for i := range cfg.Apps {
		if cfg.Apps[i].Arguments == nil {
			cfg.Apps[i].Arguments = []string{}
		}
		if cfg.Apps[i].Environment == nil {
			cfg.Apps[i].Environment = map[string]string{}
		}
	}
	if cfg.Actions == nil {
		cfg.Actions = []Action{}
	}
	if cfg.NewFiles == nil {
		cfg.NewFiles = []NewFile{}
	}
	if cfg.CommonDirs == nil {
		cfg.CommonDirs = []CommonDir{}
	}
	if cfg.PermissionDirs == nil {
		cfg.PermissionDirs = []PermissionDir{}
	}
	if cfg.QuickCommands == nil {
		cfg.QuickCommands = []QuickCommand{}
	}
}

func validateIDs(ids []string, section string, problems *[]string) {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			*problems = append(*problems, section+" contains an empty id")
			continue
		}
		if _, ok := seen[id]; ok {
			*problems = append(*problems, fmt.Sprintf("%s contains duplicate id %q", section, id))
			continue
		}
		seen[id] = struct{}{}
	}
}

func appIDs(items []OpenWithApp) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func actionIDs(items []Action) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func newFileIDs(items []NewFile) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func commonDirIDs(items []CommonDir) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func permissionDirIDs(items []PermissionDir) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func quickCommandIDs(items []QuickCommand) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}
