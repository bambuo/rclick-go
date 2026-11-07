// Package config owns the shared RClick configuration file.
//
// The FinderSync extension reads the same file through SharedConfigStore,
// so the JSON keys here are a wire contract: do not rename them without
// updating the Swift side.
package config

import "time"

const SchemaVersion = 1

type Config struct {
	SchemaVersion  int             `json:"schemaVersion"`
	Revision       int64           `json:"revision"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	Apps           []OpenWithApp   `json:"apps"`
	Actions        []Action        `json:"actions"`
	NewFiles       []NewFile       `json:"newFiles"`
	CommonDirs     []CommonDir     `json:"commonDirs"`
	PermissionDirs []PermissionDir `json:"permissionDirs"`
	QuickCommands  []QuickCommand  `json:"quickCommands"`
	Menu           MenuSettings    `json:"menu"`
	Defaults       DefaultSettings `json:"defaults"`
	UI             UISettings      `json:"ui"`
}

type OpenWithApp struct {
	ID                           string            `json:"id"`
	URL                          string            `json:"url"`
	ItemName                     string            `json:"itemName"`
	InheritFromGlobalArguments   bool              `json:"inheritFromGlobalArguments"`
	InheritFromGlobalEnvironment bool              `json:"inheritFromGlobalEnvironment"`
	Arguments                    []string          `json:"arguments"`
	Environment                  map[string]string `json:"environment"`
}

type Action struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Idx     int    `json:"idx"`
	Icon    string `json:"icon"`
}

type NewFile struct {
	ID       string `json:"id"`
	Ext      string `json:"ext"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Idx      int    `json:"idx"`
	Icon     string `json:"icon"`
	OpenApp  string `json:"openApp,omitempty"`
	Template string `json:"template,omitempty"`
}

type CommonDir struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon"`
}

type PermissionDir struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Bookmark string `json:"bookmark,omitempty"`
}

type QuickCommand struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Template  string `json:"template"`
	Enabled   bool   `json:"enabled"`
	Dangerous bool   `json:"dangerous"`
	Idx       int    `json:"idx"`
}

type MenuSettings struct {
	ShowForItems              bool   `json:"showForItems"`
	ShowForContainer          bool   `json:"showForContainer"`
	ShowForSidebar            bool   `json:"showForSidebar"`
	ShowToolbarItem           bool   `json:"showToolbarItem"`
	ShowSubMenuForApplication bool   `json:"showSubMenuForApplication"`
	ShowSubMenuForAction      bool   `json:"showSubMenuForAction"`
	CopySeparator             string `json:"copySeparator"`
}

type DefaultSettings struct {
	NewFileName                  string `json:"newFileName"`
	NewFileExtension             string `json:"newFileExtension"`
	GlobalApplicationArguments   string `json:"globalApplicationArguments"`
	GlobalApplicationEnvironment string `json:"globalApplicationEnvironment"`
}

type UISettings struct {
	ShowMenuBarExtra bool   `json:"showMenuBarExtra"`
	ShowInDock       bool   `json:"showInDock"`
	Language         string `json:"language"`
}
