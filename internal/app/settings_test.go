package app

import (
	"path/filepath"
	"testing"

	"github.com/bambuo/rclick-go/internal/config"
	"github.com/bambuo/rclick-go/internal/i18n"
	"github.com/egoist/mygo/ui"
)

func TestSettingsSections(t *testing.T) {
	sections := settingsSections()
	want := []string{"general", "apps", "actions", "newfiles", "folders", "commands"}
	if len(sections) != len(want) {
		t.Fatalf("settingsSections() returned %d sections, want %d", len(sections), len(want))
	}
	for i, section := range sections {
		if section.ID != want[i] {
			t.Errorf("section %d ID = %q, want %q", i, section.ID, want[i])
		}
		if section.LabelKey == "" || section.DescKey == "" {
			t.Errorf("section %q has empty label or description key", section.ID)
		}
	}
}

func TestCurrentSettingsSection(t *testing.T) {
	if got := currentSettingsSection("commands"); got.ID != "commands" {
		t.Errorf("currentSettingsSection(commands) = %q", got.ID)
	}
	if got := currentSettingsSection("missing"); got.ID != "general" {
		t.Errorf("currentSettingsSection(missing) = %q, want general", got.ID)
	}
}

func TestSettingsViewLayout(t *testing.T) {
	i18n.Set(i18n.Chinese)
	defer i18n.Set(i18n.English)

	store, err := config.Open(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("open config: %v", err)
	}
	a := &App{store: store}
	a.settings.section = "general"
	a.settings.draft = store.Current()
	a.settings.draft.UI.Language = "跟随系统"
	a.settings.appArgs = map[string]*string{}
	a.settings.appEnv = map[string]*string{}
	a.settings.commandNames = map[string]*string{}

	tt := ui.NewTester(a.settingsView, 880, 720)
	tt.Frame()

	for _, label := range []string{"通用", "应用", "操作", "新建文件", "文件夹", "命令"} {
		item, ok := tt.AccessByRole(ui.RoleTreeItem, label)
		if !ok {
			t.Fatalf("sidebar item %q not found", label)
		}
		if item.Bounds.X >= 208 {
			t.Errorf("sidebar item %q starts at x=%.1f, want inside the 208-wide sidebar", label, item.Bounds.X)
		}
	}
	if !tt.HasText("Finder 菜单") {
		t.Error("general section content is not visible")
	}

	appsItem, ok := tt.AccessByRole(ui.RoleTreeItem, "应用")
	if !ok {
		t.Fatal("apps sidebar item not found")
	}
	tt.ClickAt(appsItem.Bounds.X+appsItem.Bounds.W/2, appsItem.Bounds.Y+appsItem.Bounds.H/2)
	tt.Frame()
	if a.settings.section != "apps" {
		t.Fatalf("section after clicking apps = %q", a.settings.section)
	}
	if !tt.HasText("打开方式应用") {
		t.Error("apps section content is not visible")
	}
	if tt.HasText("Finder 菜单") {
		t.Error("general section still visible after switching to apps")
	}

	save, ok := tt.AccessByRole(ui.RoleButton, "保存")
	if !ok {
		t.Fatal("save button not found")
	}
	if !save.Disabled {
		t.Error("save button is enabled without changes")
	}
	a.settings.dirty = true
	tt.Frame()
	save, ok = tt.AccessByRole(ui.RoleButton, "保存")
	if !ok {
		t.Fatal("save button not found after marking dirty")
	}
	if save.Disabled {
		t.Error("save button is disabled with unsaved changes")
	}

	for _, section := range []struct {
		label string
		text  string
	}{
		{"通用", "Finder 菜单"},
		{"应用", "打开方式应用"},
		{"操作", "Finder 操作"},
		{"新建文件", "新建文件模板"},
		{"文件夹", "常用文件夹"},
		{"命令", "快捷命令"},
	} {
		item, ok := tt.AccessByRole(ui.RoleTreeItem, section.label)
		if !ok {
			t.Fatalf("sidebar item %q not found", section.label)
		}
		tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
		tt.Frame()
		if !tt.HasText(section.text) {
			t.Errorf("%s section is missing text %q", section.label, section.text)
		}
	}

	tt.SetSize(700, 560)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleTreeItem, "命令"); !ok {
		t.Error("sidebar item 命令 is missing at the minimum window size")
	}
}
