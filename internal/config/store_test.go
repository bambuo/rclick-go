package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesDefaultAndBumpsRevisionOnSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "RClick", "config.json")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config not created: %v", err)
	}

	first := store.Current()
	if first.SchemaVersion != SchemaVersion {
		t.Fatalf("schemaVersion = %d", first.SchemaVersion)
	}
	if first.Defaults.NewFileName != "Untitled" {
		t.Fatalf("newFileName = %q", first.Defaults.NewFileName)
	}

	first.Apps = append(first.Apps, OpenWithApp{
		ID:          "/Applications/Foo.app",
		URL:         "/Applications/Foo.app",
		ItemName:    "Foo",
		Arguments:   []string{"--flag"},
		Environment: map[string]string{"A": "B"},
	})
	saved, err := store.Save(first)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.Revision != 1 {
		t.Fatalf("revision = %d, want 1", saved.Revision)
	}
	if saved.UpdatedAt.IsZero() {
		t.Fatal("updatedAt not set")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}

	again, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	loaded := again.Current()
	if len(loaded.Apps) != len(first.Apps) {
		t.Fatalf("apps = %d, want %d", len(loaded.Apps), len(first.Apps))
	}
	if got := loaded.Apps[len(loaded.Apps)-1].Environment["A"]; got != "B" {
		t.Fatalf("environment roundtrip = %q", got)
	}
}

func TestDecodeFillsMenuDefaults(t *testing.T) {
	cfg, err := decode([]byte(`{"schemaVersion":1,"revision":3,"apps":[{"id":"a","url":"/A.app","itemName":"A"}],"actions":[],"newFiles":[],"commonDirs":[],"permissionDirs":[],"quickCommands":[]}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !cfg.Menu.ShowForItems || !cfg.Menu.ShowForContainer || !cfg.Menu.ShowToolbarItem {
		t.Fatalf("menu defaults not applied: %+v", cfg.Menu)
	}
	if cfg.Defaults.NewFileName != "Untitled" {
		t.Fatalf("newFileName = %q", cfg.Defaults.NewFileName)
	}
	if cfg.Apps == nil || cfg.QuickCommands == nil {
		t.Fatal("nil slices should be normalized")
	}
	if cfg.Apps[0].Arguments == nil || cfg.Apps[0].Environment == nil {
		t.Fatal("nested app collections should be non-nil for the Swift decoder")
	}
}

func TestSwiftWireKeys(t *testing.T) {
	cfg := NewDefault()
	cfg.PermissionDirs = []PermissionDir{{ID: "home", URL: "/Users/test"}}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"schemaVersion", "revision", "updatedAt", "apps", "actions", "newFiles",
		"commonDirs", "permissionDirs", "quickCommands", "menu", "defaults", "ui",
	} {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}

	var menu map[string]json.RawMessage
	if err := json.Unmarshal(raw["menu"], &menu); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"showForItems", "showForContainer", "showForSidebar", "showToolbarItem",
		"showSubMenuForApplication", "showSubMenuForAction", "copySeparator",
	} {
		if _, ok := menu[key]; !ok {
			t.Errorf("missing menu key %q", key)
		}
	}
}

func TestNormalizeReindexes(t *testing.T) {
	cfg := Config{
		SchemaVersion: SchemaVersion,
		Actions: []Action{
			{ID: "b", Idx: 5},
			{ID: "a", Idx: 1},
		},
		NewFiles: []NewFile{
			{ID: "n2", Idx: 9},
			{ID: "n1", Idx: 2},
		},
	}
	Normalize(&cfg)
	if cfg.Actions[0].ID != "a" || cfg.Actions[0].Idx != 0 || cfg.Actions[1].Idx != 1 {
		t.Fatalf("actions not normalized: %+v", cfg.Actions)
	}
	if cfg.NewFiles[0].ID != "n1" || cfg.NewFiles[0].Idx != 0 {
		t.Fatalf("newFiles not normalized: %+v", cfg.NewFiles)
	}
}

func TestValidateRejectsDuplicateIDs(t *testing.T) {
	cfg := Config{
		SchemaVersion: SchemaVersion,
		Actions:       []Action{{ID: "same"}, {ID: "same"}},
	}
	if err := Validate(&cfg); err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestOpenKeepsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := Open(path)
	if err == nil {
		t.Fatal("expected decode error")
	}
	if store == nil {
		t.Fatal("store should still allow defaults")
	}
	if len(store.Current().Actions) == 0 {
		t.Fatal("fallback has no default actions")
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "{not json" {
		t.Fatal("corrupt file was overwritten")
	}
}
