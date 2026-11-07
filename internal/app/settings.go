package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/bambuo/rclick-go/internal/config"
	"github.com/bambuo/rclick-go/internal/i18n"
	"github.com/bambuo/rclick-go/internal/services"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type settingsState struct {
	section      string
	draft        config.Config
	dirty        bool
	status       string
	appArgs      map[string]*string
	appEnv       map[string]*string
	commandNames map[string]*string

	launchAtLogin bool
	newFileName   string
	newFileExt    string
}

type settingsSection struct {
	ID       string
	LabelKey string
	DescKey  string
}

func settingsSections() []settingsSection {
	return []settingsSection{
		{
			ID:       "general",
			LabelKey: "General",
			DescKey:  "Control where RClick appears in Finder and how new files are named.",
		},
		{
			ID:       "apps",
			LabelKey: "Apps",
			DescKey:  "Manage the apps in Open With, with their own arguments and environment.",
		},
		{
			ID:       "actions",
			LabelKey: "Actions",
			DescKey:  "Choose which file actions appear in the Finder context menu.",
		},
		{
			ID:       "newfiles",
			LabelKey: "New Files",
			DescKey:  "Manage the file types offered by New File and their extensions.",
		},
		{
			ID:       "folders",
			LabelKey: "Folders",
			DescKey:  "Configure favorite folders and the folders where the RClick menu appears.",
		},
		{
			ID:       "commands",
			LabelKey: "Commands",
			DescKey:  "Configure quick commands and whether they ask before running.",
		},
	}
}

func currentSettingsSection(id string) settingsSection {
	sections := settingsSections()
	for _, section := range sections {
		if section.ID == id {
			return section
		}
	}
	return sections[0]
}

func (a *App) markDirty() {
	a.settings.dirty = true
}

func (a *App) settingsView(c *ui.Context) {
	t := c.Theme()
	if a.settings.section == "" {
		a.settings.section = "general"
	}
	section := currentSettingsSection(a.settings.section)

	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Width(208).Shrink(0).Background(t.Surface).Children(func() {
			ui.Column(c).Padding(16, 16, 8).Gap(2).Children(func() {
				ui.Text(c, "RClick").FontSize(17).Bold()
				ui.Text(c, i18n.T("Finder context menu")).FontSize(11).TextColor(t.TextMuted)
			})
			ui.Sidebar(c, &a.settings.section, func() {
				for _, item := range settingsSections() {
					ui.SidebarItem(c, item.ID, settingsIcons[item.ID], i18n.T(item.LabelKey))
				}
			}).Grow(1).Label(i18n.T("Settings sections"))
			ui.Text(c, i18n.T("Saved changes are written to config.json and picked up by the Finder extension.")).
				FontSize(11).TextColor(t.TextMuted).Padding(8, 16, 14)
		})

		ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
			ui.Column(c).Padding(20, 24, 8).Gap(4).Children(func() {
				ui.Text(c, i18n.T(section.LabelKey)).FontSize(20).Bold()
				ui.Text(c, i18n.T(section.DescKey)).FontSize(12).TextColor(t.TextMuted)
			})

			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Padding(0, 24, 18).Gap(12).Children(func() {
					switch section.ID {
					case "apps":
						a.appsTab(c)
					case "actions":
						a.actionsTab(c)
					case "newfiles":
						a.newFilesTab(c)
					case "folders":
						a.foldersTab(c)
					case "commands":
						a.commandsTab(c)
					default:
						a.generalTab(c)
					}
				})
			})

			status := a.settings.status
			if status == "" {
				status = i18n.T("Saved")
			}
			ui.Row(c).Padding(10, 24).Gap(8).AlignItems(ui.Center).
				BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
				if a.settings.dirty {
					ui.Text(c, i18n.T("Unsaved changes")).FontSize(11).TextColor(t.Accent)
					if status != i18n.T("Saved") {
						ui.Text(c, "·").FontSize(11).TextColor(t.TextMuted)
					}
				}
				if !a.settings.dirty || status != i18n.T("Saved") {
					ui.Text(c, status).FontSize(11).TextColor(t.TextMuted)
				}
				ui.Spacer(c)
				if ui.Button(c, i18n.T("Revert")).Disabled(!a.settings.dirty).Clicked() {
					a.revertSettings()
				}
				if ui.PrimaryButton(c, i18n.T("Save")).Disabled(!a.settings.dirty).Clicked() {
					a.saveSettings()
				}
			})
		})
	})
}

func (a *App) generalTab(c *ui.Context) {
	t := c.Theme()
	draft := &a.settings.draft

	ui.Form(c, func() {
		ui.Fieldset(c, i18n.T("Finder Menu"), func() {
			a.switchField(c, i18n.T("Selected items"), &draft.Menu.ShowForItems)
			a.switchField(c, i18n.T("Folder background"), &draft.Menu.ShowForContainer)
			a.switchField(c, i18n.T("Sidebar"), &draft.Menu.ShowForSidebar)
			a.switchField(c, i18n.T("Toolbar item"), &draft.Menu.ShowToolbarItem)
		})

		ui.Fieldset(c, i18n.T("Defaults"), func() {
			ui.Field(c, i18n.T("New file name"), func() {
				if ui.TextInput(c, &draft.Defaults.NewFileName).Changed() {
					a.markDirty()
				}
			})
			ui.Field(c, i18n.T("New file extension"), func() {
				if ui.TextInput(c, &draft.Defaults.NewFileExtension).Changed() {
					a.markDirty()
				}
			})
			ui.Field(c, i18n.T("Copy separator"), func() {
				if ui.TextInput(c, &draft.Menu.CopySeparator).Changed() {
					a.markDirty()
				}
			}).Description(i18n.T("Leave empty for a single space."))
		})

		ui.Fieldset(c, i18n.T("Open With"), func() {
			ui.Field(c, i18n.T("Global arguments"), func() {
				if ui.TextInput(c, &draft.Defaults.GlobalApplicationArguments).Changed() {
					a.markDirty()
				}
			}).Description(i18n.T("Space-separated arguments added to every application."))
			ui.Field(c, i18n.T("Global environment"), func() {
				if ui.TextInput(c, &draft.Defaults.GlobalApplicationEnvironment).Changed() {
					a.markDirty()
				}
			}).Description(i18n.T("KEY=value pairs separated by spaces."))
		})

		ui.Fieldset(c, i18n.T("Application"), func() {
			ui.Field(c, i18n.T("Language"), func() {
				if ui.Select(c, &draft.UI.Language, languageOptions()).Changed() {
					a.markDirty()
				}
			})
			ui.Field(c, i18n.T("Start at login"), func() {
				if ui.Switch(c, &a.settings.launchAtLogin).Changed() {
					if err := mygo.App.SetOpenAtLogin(a.settings.launchAtLogin); err != nil {
						a.settings.status = i18n.Sprintf("Login item: %s", err.Error())
						a.settings.launchAtLogin = currentOpenAtLogin()
					} else {
						a.settings.status = i18n.T("Login item updated")
					}
				}
			})
			ui.Field(c, i18n.T("Show in Dock"), func() {
				if ui.Switch(c, &draft.UI.ShowInDock).Changed() {
					a.markDirty()
				}
			}).Description(i18n.T("Applied when you save."))
		})
	})

	ui.Text(c, i18n.Sprintf("The configuration lives at %s", a.store.Path())).FontSize(11).TextColor(t.TextMuted)
}

func (a *App) appsTab(c *ui.Context) {
	t := c.Theme()
	draft := &a.settings.draft

	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, i18n.T("Open With applications")).FontSize(14).Bold()
		ui.Spacer(c)
		if ui.Button(c, i18n.T("Add Application…")).Clicked() {
			a.pickApplications()
		}
	})

	if len(draft.Apps) == 0 {
		ui.Text(c, i18n.T("No applications yet.")).TextColor(t.TextMuted)
		return
	}

	remove := -1
	for i := range draft.Apps {
		application := &draft.Apps[i]
		key := application.ID
		if a.settings.appArgs[key] == nil {
			a.settings.appArgs[key] = new(string)
		}
		if a.settings.appEnv[key] == nil {
			a.settings.appEnv[key] = new(string)
		}
		ui.Column(c).Gap(8).Padding(12, 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				if ui.TextInput(c.Key("app-name-"+key), &application.ItemName).Grow(1).Changed() {
					a.markDirty()
				}
				if ui.Button(c, i18n.T("Remove")).Clicked() {
					remove = i
				}
			})
			ui.Text(c, application.URL).FontSize(11).TextColor(t.TextMuted)

			ui.Row(c).Gap(18).Children(func() {
				if ui.Switch(c, &application.InheritFromGlobalArguments).Label(i18n.T("Inherit global arguments")).Changed() {
					a.markDirty()
				}
				ui.Text(c, i18n.T("Inherit arguments")).FontSize(12)
				if ui.Switch(c, &application.InheritFromGlobalEnvironment).Label(i18n.T("Inherit global environment")).Changed() {
					a.markDirty()
				}
				ui.Text(c, i18n.T("Inherit environment")).FontSize(12)
			})

			ui.Field(c, i18n.T("Arguments"), func() {
				if ui.TextInput(c.Key("app-args-"+key), a.settings.appArgs[key]).Changed() {
					application.Arguments = strings.Fields(*a.settings.appArgs[key])
					a.markDirty()
				}
			})
			ui.Field(c, i18n.T("Environment"), func() {
				if ui.TextInput(c.Key("app-env-"+key), a.settings.appEnv[key]).Changed() {
					application.Environment = decodeEnvironment(*a.settings.appEnv[key])
					a.markDirty()
				}
			}).Description(i18n.T("KEY=value pairs separated by spaces."))
		})
	}
	if remove >= 0 {
		draft.Apps = append(draft.Apps[:remove], draft.Apps[remove+1:]...)
		a.markDirty()
	}
}

func (a *App) actionsTab(c *ui.Context) {
	t := c.Theme()
	draft := &a.settings.draft

	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, i18n.T("Finder actions")).FontSize(14).Bold()
		ui.Spacer(c)
		if ui.Button(c, i18n.T("Reset Actions")).Clicked() {
			draft.Actions = config.DefaultActions()
			a.markDirty()
		}
	})

	for i := range draft.Actions {
		action := &draft.Actions[i]
		name := i18n.ActionName(action.ID, action.Name)
		ui.Row(c).Gap(10).Padding(10, 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			if ui.Switch(c, &action.Enabled).Label(name).Changed() {
				a.markDirty()
			}
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, name).FontSize(13)
				ui.Text(c, action.ID).FontSize(11).TextColor(t.TextMuted)
			})
		})
	}
}

func (a *App) newFilesTab(c *ui.Context) {
	draft := &a.settings.draft

	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, i18n.T("New File templates")).FontSize(14).Bold()
		ui.Spacer(c)
		if ui.Button(c, i18n.T("Reset Templates")).Clicked() {
			draft.NewFiles = config.DefaultNewFiles()
			a.markDirty()
		}
	})

	remove := -1
	for i := range draft.NewFiles {
		file := &draft.NewFiles[i]
		ui.Row(c).Gap(10).Padding(10, 0).BorderWidth(0, 0, 1, 0).BorderColor(c.Theme().Border).Children(func() {
			if ui.Switch(c, &file.Enabled).Label(i18n.NewFileTypeName(file.ID, file.Name)).Changed() {
				a.markDirty()
			}
			if ui.TextInput(c.Key("newfile-name-"+file.ID), &file.Name).Grow(1).MinWidth(120).Changed() {
				a.markDirty()
			}
			if ui.TextInput(c.Key("newfile-ext-"+file.ID), &file.Ext).Width(100).Changed() {
				a.markDirty()
			}
			ui.Spacer(c)
			if ui.Button(c, i18n.T("Remove")).Clicked() {
				remove = i
			}
		})
	}
	if remove >= 0 {
		draft.NewFiles = append(draft.NewFiles[:remove], draft.NewFiles[remove+1:]...)
		a.markDirty()
	}

	ui.Divider(c)
	ui.Text(c, i18n.T("Add a template")).FontSize(13).Bold()
	ui.Row(c).Gap(8).Children(func() {
		ui.TextInput(c.Key("newfile-new-name"), &a.settings.newFileName).Placeholder(i18n.T("Name")).Grow(1).MinWidth(120)
		ui.TextInput(c.Key("newfile-new-ext"), &a.settings.newFileExt).Placeholder(i18n.T("Extension")).Width(110)
		if ui.Button(c, i18n.T("Add")).Clicked() {
			a.addNewFileType()
		}
	})
}

func (a *App) foldersTab(c *ui.Context) {
	t := c.Theme()
	draft := &a.settings.draft

	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, i18n.T("Favorite Folders")).FontSize(14).Bold()
		ui.Spacer(c)
		if ui.Button(c, i18n.T("Add Folder…")).Clicked() {
			a.pickFolders(false)
		}
	})
	if len(draft.CommonDirs) == 0 {
		ui.Text(c, i18n.T("No favorite folders yet.")).TextColor(t.TextMuted)
	}

	removeCommon := -1
	for i := range draft.CommonDirs {
		dir := &draft.CommonDirs[i]
		ui.Row(c).Gap(10).Padding(10, 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			if ui.TextInput(c.Key("dir-name-"+dir.ID), &dir.Name).Grow(1).Changed() {
				a.markDirty()
			}
			ui.Text(c, dir.URL).FontSize(11).TextColor(t.TextMuted).Grow(1)
			if ui.Button(c, i18n.T("Remove")).Clicked() {
				removeCommon = i
			}
		})
	}
	if removeCommon >= 0 {
		draft.CommonDirs = append(draft.CommonDirs[:removeCommon], draft.CommonDirs[removeCommon+1:]...)
		a.markDirty()
	}

	ui.Divider(c)

	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, i18n.T("Watched Folders")).FontSize(14).Bold()
		ui.Spacer(c)
		if ui.Button(c, i18n.T("Add Folder…")).Clicked() {
			a.pickFolders(true)
		}
	})
	ui.Text(c, i18n.T("Folders where the RClick menu appears. Save, then refresh the Finder menu.")).
		FontSize(11).TextColor(t.TextMuted)

	removeWatched := -1
	for i := range draft.PermissionDirs {
		dir := &draft.PermissionDirs[i]
		ui.Row(c).Gap(10).Padding(10, 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, dir.URL).FontSize(12).Grow(1)
			if ui.Button(c, i18n.T("Remove")).Clicked() {
				removeWatched = i
			}
		})
	}
	if removeWatched >= 0 {
		draft.PermissionDirs = append(draft.PermissionDirs[:removeWatched], draft.PermissionDirs[removeWatched+1:]...)
		a.markDirty()
	}
}

func (a *App) commandsTab(c *ui.Context) {
	t := c.Theme()
	draft := &a.settings.draft

	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, i18n.T("Quick Commands")).FontSize(14).Bold()
		ui.Spacer(c)
		if ui.Button(c, i18n.T("Reset Commands")).Clicked() {
			draft.QuickCommands = config.DefaultQuickCommands()
			a.markDirty()
		}
		if ui.Button(c, i18n.T("Add Command")).Clicked() {
			a.addQuickCommand()
		}
	})

	remove := -1
	for i := range draft.QuickCommands {
		command := &draft.QuickCommands[i]
		if a.settings.commandNames[command.ID] == nil {
			name := i18n.CommandName(command.ID, command.Name)
			a.settings.commandNames[command.ID] = &name
		}
		displayName := *a.settings.commandNames[command.ID]
		ui.Column(c).Gap(8).Padding(12, 0).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				if ui.Switch(c, &command.Enabled).Label(displayName).Changed() {
					a.markDirty()
				}
				if ui.TextInput(c.Key("qc-name-"+command.ID), a.settings.commandNames[command.ID]).Grow(1).Changed() {
					command.Name = *a.settings.commandNames[command.ID]
					a.markDirty()
				}
				ui.Spacer(c)
				if ui.Button(c, i18n.T("Remove")).Clicked() {
					remove = i
				}
			})
			ui.Row(c).Gap(18).Children(func() {
				if ui.Switch(c, &command.Dangerous).Label(i18n.T("Ask before running")).Changed() {
					a.markDirty()
				}
				ui.Text(c, i18n.T("Ask before running")).FontSize(12)
			})
			if ui.TextArea(c.Key("qc-template-"+command.ID), &command.Template).Height(62).Changed() {
				a.markDirty()
			}
			ui.Text(c, i18n.T("Placeholders: {path} {name} {dir} {ext} {name_no_ext} {all}")).
				FontSize(11).TextColor(t.TextMuted)
		})
	}
	if remove >= 0 {
		draft.QuickCommands = append(draft.QuickCommands[:remove], draft.QuickCommands[remove+1:]...)
		a.markDirty()
	}
}

func (a *App) switchField(c *ui.Context, label string, value *bool) {
	ui.Field(c, label, func() {
		if ui.Switch(c, value).Changed() {
			a.markDirty()
		}
	})
}

func (a *App) addNewFileType() {
	name := strings.TrimSpace(a.settings.newFileName)
	if name == "" {
		a.settings.status = i18n.T("Enter a template name")
		return
	}
	ext := services.NormalizeExtension(strings.TrimSpace(a.settings.newFileExt))
	draft := &a.settings.draft
	id := fmt.Sprintf("newfile-%d", time.Now().UnixNano())
	draft.NewFiles = append(draft.NewFiles, config.NewFile{
		ID:      id,
		Ext:     ext,
		Name:    name,
		Enabled: true,
		Idx:     len(draft.NewFiles),
		Icon:    "doc",
	})
	a.settings.newFileName = ""
	a.settings.newFileExt = ""
	a.markDirty()
}

func (a *App) addQuickCommand() {
	draft := &a.settings.draft
	id := fmt.Sprintf("qc-%d", time.Now().UnixNano())
	draft.QuickCommands = append(draft.QuickCommands, config.QuickCommand{
		ID:       id,
		Name:     i18n.T("New Command"),
		Icon:     "terminal",
		Template: `echo "{path}"`,
		Enabled:  true,
		Idx:      len(draft.QuickCommands),
	})
	name := i18n.T("New Command")
	a.settings.commandNames[id] = &name
	a.markDirty()
}

func languageOptions() []string {
	return []string{i18n.T("System"), "English", "简体中文"}
}

// languageLabel maps a stored language value to the label of its option.
func languageLabel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "system", "跟随系统":
		return i18n.T("System")
	case "en", "english":
		return "English"
	case "zh", "zh-hans", "简体中文", "中文":
		return "简体中文"
	default:
		return value
	}
}

// canonicalLanguage stores a stable value in config.json.
func canonicalLanguage(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "", "system", "跟随系统":
		return "system"
	case "en", "english":
		return "en"
	case "zh", "zh-hans", "简体中文", "中文":
		return "zh-Hans"
	default:
		return label
	}
}
