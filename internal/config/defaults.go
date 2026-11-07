package config

import (
	"os"
	"time"
)

const (
	TerminalPath = "/System/Applications/Utilities/Terminal.app"
	VSCodePath   = "/Applications/Visual Studio Code.app"
)

// NewDefault returns the configuration a fresh install starts with.
// Only apps that exist on this Mac are added, matching the original app.
func NewDefault() Config {
	var apps []OpenWithApp
	if isDir(TerminalPath) {
		apps = append(apps, newDefaultApp(TerminalPath, "Terminal"))
	}
	if isDir(VSCodePath) {
		apps = append(apps, newDefaultApp(VSCodePath, "Visual Studio Code"))
	}

	return Config{
		SchemaVersion: SchemaVersion,
		UpdatedAt:     time.Now().UTC(),
		Apps:          apps,
		Actions:       DefaultActions(),
		NewFiles:      DefaultNewFiles(),
		QuickCommands: DefaultQuickCommands(),
		Menu: MenuSettings{
			ShowForItems:              true,
			ShowForContainer:          true,
			ShowForSidebar:            true,
			ShowToolbarItem:           true,
			ShowSubMenuForApplication: false,
			ShowSubMenuForAction:      false,
			CopySeparator:             "",
		},
		Defaults: DefaultSettings{
			NewFileName: "Untitled",
		},
		UI: UISettings{
			ShowMenuBarExtra: true,
		},
	}
}

func DefaultActions() []Action {
	return []Action{
		{ID: "copy-path", Name: "Copy Path", Enabled: true, Idx: 0, Icon: "doc.on.doc"},
		{ID: "delete-direct", Name: "Delete Direct", Enabled: true, Idx: 1, Icon: "trash"},
		{ID: "hide", Name: "Hide", Enabled: true, Idx: 2, Icon: "eye.slash"},
		{ID: "unhide", Name: "Unhide", Enabled: true, Idx: 3, Icon: "eye"},
		{ID: "airdrop", Name: "AirDrop", Enabled: true, Idx: 4, Icon: "paperplane"},
	}
}

func DefaultNewFiles() []NewFile {
	return []NewFile{
		{ID: "newfile-json", Ext: ".json", Name: "JSON", Enabled: true, Idx: 0, Icon: "icon-file-json"},
		{ID: "newfile-txt", Ext: ".txt", Name: "TXT", Enabled: true, Idx: 1, Icon: "icon-file-txt"},
		{ID: "newfile-md", Ext: ".md", Name: "Markdown", Enabled: true, Idx: 2, Icon: "icon-file-md"},
		{ID: "newfile-docx", Ext: ".docx", Name: "DOCX", Enabled: true, Idx: 3, Icon: "icon-file-docx"},
		{ID: "newfile-pptx", Ext: ".pptx", Name: "PPTX", Enabled: true, Idx: 4, Icon: "icon-file-pptx"},
		{ID: "newfile-xlsx", Ext: ".xlsx", Name: "XLSX", Enabled: true, Idx: 5, Icon: "icon-file-xlsx"},
	}
}

func DefaultQuickCommands() []QuickCommand {
	return []QuickCommand{
		{ID: "qc-remove-quarantine", Name: "Remove Quarantine", Icon: "shield.slash", Template: `xattr -cr "{path}"`, Enabled: true, Dangerous: true, Idx: 0},
		{ID: "qc-make-executable", Name: "Make Executable", Icon: "hammer", Template: `chmod +x "{path}"`, Enabled: true, Idx: 1},
		{ID: "qc-strip-code-sign", Name: "Strip Code Sign", Icon: "signature", Template: `codesign --remove-signature "{path}"`, Enabled: true, Dangerous: true, Idx: 2},
		{ID: "qc-show-file-info", Name: "Show File Info", Icon: "ellipsis.curlybraces", Template: `file "{path}"`, Enabled: true, Idx: 3},
		{ID: "qc-zip-file", Name: "Zip File", Icon: "rectangle.compress.vertical", Template: `zip -r "{name_no_ext}.zip" "{path}"`, Enabled: true, Idx: 4},
	}
}

func newDefaultApp(path, name string) OpenWithApp {
	return OpenWithApp{
		ID:                           path,
		URL:                          path,
		ItemName:                     name,
		InheritFromGlobalArguments:   true,
		InheritFromGlobalEnvironment: true,
		Arguments:                    []string{},
		Environment:                  map[string]string{},
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
