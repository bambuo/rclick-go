package app

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/bambuo/rclick-go/internal/config"
	"github.com/bambuo/rclick-go/internal/i18n"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) trayMenu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{
			Label:       i18n.T("Settings…"),
			Accelerator: "CmdOrCtrl+,",
			Click:       func(*mygo.MenuItem, *mygo.Window) { a.showSettings() },
		},
		{
			Label: i18n.T("Refresh Finder Menu"),
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.sendRunning()
				a.settings.status = i18n.T("Finder menu refreshed")
				a.invalidateSettings()
			},
		},
		mygo.Separator(),
		{
			Label:       i18n.T("Quit RClick"),
			Accelerator: "CmdOrCtrl+Q",
			Click:       func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() },
		},
	})
}

func (a *App) createTray() {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:             "rclick",
		Icon:           trayIcon,
		IconIsTemplate: true,
		ToolTip:        "RClick",
		Menu:           a.trayMenu(),
	})
	if err != nil {
		log.Printf("create tray: %v", err)
		return
	}
	a.tray = tray
}

func (a *App) rebuildTrayMenu() {
	if a.tray != nil {
		a.tray.SetMenu(a.trayMenu())
	}
}

func (a *App) createSettingsWindow() {
	a.window = mygo.NewWindow(mygo.WindowOptions{
		Title:     i18n.T("RClick Settings"),
		Width:     760,
		Height:    680,
		MinWidth:  620,
		MinHeight: 480,
		Hidden:    true,
		Content:   ui.View(a.settingsView),
	})
	a.window.OnClose(func(e *mygo.CloseEvent) {
		if a.quitting {
			return
		}
		e.PreventDefault()
		a.window.Hide()
	})
}

func (a *App) showSettings() {
	if a.window == nil {
		return
	}
	if !a.settings.dirty {
		a.refreshSettings()
	}
	a.window.Show()
	a.window.Focus()
	mygo.App.Focus()
}

func (a *App) invalidateSettings() {
	if a.window == nil || a.window.IsDestroyed() {
		return
	}
	a.window.Invalidate()
}

func (a *App) refreshSettings() {
	a.settings.draft = a.store.Current()
	a.settings.draft.UI.Language = languageLabel(a.settings.draft.UI.Language)
	a.settings.dirty = false
	a.settings.launchAtLogin = currentOpenAtLogin()
	a.settings.appArgs = map[string]*string{}
	a.settings.appEnv = map[string]*string{}
	a.settings.commandNames = map[string]*string{}
	for _, application := range a.settings.draft.Apps {
		args := strings.Join(application.Arguments, " ")
		env := encodeEnvironment(application.Environment)
		a.settings.appArgs[application.ID] = &args
		a.settings.appEnv[application.ID] = &env
	}
	for _, command := range a.settings.draft.QuickCommands {
		name := i18n.CommandName(command.ID, command.Name)
		a.settings.commandNames[command.ID] = &name
	}
}

func (a *App) saveSettings() {
	a.settings.draft.UI.Language = canonicalLanguage(a.settings.draft.UI.Language)
	saved, err := a.store.Save(a.settings.draft)
	if err != nil {
		a.settings.status = i18n.Sprintf("Save failed: %s", err.Error())
		return
	}

	i18n.Apply(saved.UI.Language)
	a.refreshSettings()
	a.settings.status = i18n.T("Saved")
	mygo.App.SetActivationPolicy(activationPolicy(saved.UI.ShowInDock))
	if a.window != nil && !a.window.IsDestroyed() {
		a.window.SetTitle(i18n.T("RClick Settings"))
	}
	a.rebuildTrayMenu()
	a.sendRunning()
	a.invalidateSettings()
}

func (a *App) revertSettings() {
	a.refreshSettings()
	i18n.Apply(a.store.Current().UI.Language)
	if a.window != nil && !a.window.IsDestroyed() {
		a.window.SetTitle(i18n.T("RClick Settings"))
	}
	a.rebuildTrayMenu()
	a.settings.status = i18n.T("Reverted to the saved configuration")
}

func (a *App) pickApplications() {
	if a.window == nil {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:   i18n.T("Choose an Application"),
			Filters: []mygo.FileFilter{{Name: i18n.T("Applications"), Extensions: []string{"app"}}},
			Parent:  a.window,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.window.Update(func() {
			for _, path := range paths {
				a.addApplication(path)
			}
		})
	}()
}

func (a *App) pickFolders(permission bool) {
	if a.window == nil {
		return
	}
	title := i18n.T("Choose Favorite Folders")
	if permission {
		title = i18n.T("Choose Watched Folders")
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:     title,
			Directory: true,
			Multiple:  true,
			Parent:    a.window,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.window.Update(func() {
			for _, path := range paths {
				if permission {
					a.addPermissionDir(path)
				} else {
					a.addCommonDir(path)
				}
			}
		})
	}()
}

func (a *App) addApplication(path string) {
	for _, existing := range a.settings.draft.Apps {
		if existing.ID == path {
			a.settings.status = i18n.T("Application already added")
			return
		}
	}
	name := strings.TrimSuffix(filepath.Base(path), ".app")
	a.settings.draft.Apps = append(a.settings.draft.Apps, config.OpenWithApp{
		ID:                           path,
		URL:                          path,
		ItemName:                     name,
		InheritFromGlobalArguments:   true,
		InheritFromGlobalEnvironment: true,
		Arguments:                    []string{},
		Environment:                  map[string]string{},
	})
	a.settings.appArgs[path] = new(string)
	a.settings.appEnv[path] = new(string)
	a.settings.dirty = true
}

func (a *App) addCommonDir(path string) {
	for _, existing := range a.settings.draft.CommonDirs {
		if existing.URL == path {
			a.settings.status = i18n.T("Folder already added")
			return
		}
	}
	a.settings.draft.CommonDirs = append(a.settings.draft.CommonDirs, config.CommonDir{
		ID:   path,
		Name: filepath.Base(path),
		URL:  path,
		Icon: "folder",
	})
	a.settings.dirty = true
}

func (a *App) addPermissionDir(path string) {
	for _, existing := range a.settings.draft.PermissionDirs {
		if existing.URL == path {
			a.settings.status = i18n.T("Folder already watched")
			return
		}
	}
	a.settings.draft.PermissionDirs = append(a.settings.draft.PermissionDirs, config.PermissionDir{
		ID:  path,
		URL: path,
	})
	a.settings.dirty = true
}

func currentOpenAtLogin() (enabled bool) {
	defer func() {
		if recover() != nil {
			enabled = false
		}
	}()
	return mygo.App.OpenAtLogin()
}

func encodeEnvironment(environment map[string]string) string {
	if len(environment) == 0 {
		return ""
	}
	parts := make([]string, 0, len(environment))
	for key, value := range environment {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, " ")
}

func decodeEnvironment(text string) map[string]string {
	out := map[string]string{}
	for _, token := range strings.Fields(text) {
		key, value, ok := strings.Cut(token, "=")
		if !ok || key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	return out
}
