// Package i18n keeps the Go UI in the user's language. The Finder
// extension localizes itself with its own .lproj resources.
package i18n

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Lang string

const (
	English Lang = "en"
	Chinese Lang = "zh-Hans"
)

var (
	mu      sync.RWMutex
	current = English
)

// Apply selects the language from a config value ("", "system", "en",
// "zh-Hans", or the labels shown in settings) and returns the result.
func Apply(setting string) Lang {
	lang := parse(setting)
	if lang == "" {
		lang = SystemLanguage()
	}
	Set(lang)
	return lang
}

func Set(lang Lang) {
	mu.Lock()
	current = lang
	mu.Unlock()
}

func Current() Lang {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func T(key string) string {
	if Current() != Chinese {
		return key
	}
	if value, ok := zh[key]; ok {
		return value
	}
	return key
}

func Sprintf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

// SystemLanguage reads the first preferred language from macOS.
func SystemLanguage() Lang {
	if out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output(); err == nil {
		if tag := firstQuoted(string(out)); tag != "" {
			return parse(tag)
		}
	}
	return parse(os.Getenv("LANG"))
}

func parse(tag string) Lang {
	value := strings.ToLower(strings.TrimSpace(tag))
	switch {
	case value == "" || value == "system" || strings.Contains(tag, "跟随系统") || strings.Contains(tag, "系統"):
		return ""
	case strings.HasPrefix(value, "zh") || strings.Contains(tag, "中文") || strings.Contains(tag, "简体"):
		return Chinese
	case strings.HasPrefix(value, "en") || strings.Contains(tag, "English"):
		return English
	default:
		return English
	}
}

func firstQuoted(text string) string {
	start := strings.Index(text, "\"")
	if start < 0 {
		return ""
	}
	rest := text[start+1:]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// ActionName localizes the built-in actions without changing their IDs.
func ActionName(id, fallback string) string {
	if Current() != Chinese {
		return fallback
	}
	switch id {
	case "copy-path":
		return "复制路径"
	case "delete-direct":
		return "直接删除"
	case "hide":
		return "隐藏"
	case "unhide":
		return "取消隐藏"
	case "airdrop":
		return "隔空投送"
	}
	return fallback
}

// CommandName localizes the default quick commands.
func CommandName(id, fallback string) string {
	if Current() != Chinese {
		return fallback
	}
	switch id {
	case "qc-remove-quarantine":
		return "移除隔离属性"
	case "qc-make-executable":
		return "设为可执行"
	case "qc-strip-code-sign":
		return "移除代码签名"
	case "qc-show-file-info":
		return "显示文件信息"
	case "qc-zip-file":
		return "压缩文件"
	}
	return fallback
}

// NewFileTypeName localizes the default file types.
func NewFileTypeName(id, fallback string) string {
	if Current() != Chinese {
		return fallback
	}
	switch id {
	case "newfile-json":
		return "JSON"
	case "newfile-txt":
		return "TXT"
	case "newfile-md":
		return "Markdown"
	case "newfile-docx":
		return "Word 文档"
	case "newfile-pptx":
		return "演示文稿"
	case "newfile-xlsx":
		return "电子表格"
	}
	return fallback
}

var zh = map[string]string{
	// Tray
	"Settings…":             "设置…",
	"Refresh Finder Menu":   "刷新 Finder 菜单",
	"Quit RClick":           "退出 RClick",
	"Finder menu refreshed": "Finder 菜单已刷新",

	// Window
	"RClick Settings":     "RClick 设置",
	"Finder context menu": "Finder 右键菜单",
	"Settings sections":   "设置分区",
	"Unsaved changes":     "未保存更改",
	"Control where RClick appears in Finder and how new files are named.":       "控制 RClick 在 Finder 中出现的位置，以及新建文件的默认命名。",
	"Manage the apps in Open With, with their own arguments and environment.":   "管理“打开方式”中的应用，以及各应用的参数和环境变量。",
	"Choose which file actions appear in the Finder context menu.":              "选择在 Finder 右键菜单中显示的文件操作。",
	"Manage the file types offered by New File and their extensions.":           "管理“新建文件”提供的文件类型与扩展名。",
	"Configure favorite folders and the folders where the RClick menu appears.": "配置常用文件夹，以及显示 RClick 菜单的文件夹。",
	"Configure quick commands and whether they ask before running.":             "配置快捷命令，以及运行前是否确认。",
	"Revert":    "还原",
	"Save":      "保存",
	"General":   "通用",
	"Apps":      "应用",
	"Actions":   "操作",
	"New Files": "新建文件",
	"Folders":   "文件夹",
	"Commands":  "命令",
	"Saved changes are written to config.json and picked up by the Finder extension.": "保存后会写入 config.json，Finder 扩展将自动读取。",

	// General
	"Finder Menu":                     "Finder 菜单",
	"Selected items":                  "选中项目",
	"Folder background":               "文件夹背景",
	"Sidebar":                         "边栏",
	"Toolbar item":                    "工具栏项目",
	"Defaults":                        "默认值",
	"New file name":                   "新建文件名称",
	"New file extension":              "新建文件扩展名",
	"Copy separator":                  "复制分隔符",
	"Leave empty for a single space.": "留空则使用单个空格。",
	"Open With":                       "打开方式",
	"Global arguments":                "全局参数",
	"Space-separated arguments added to every application.": "添加到每个应用的参数，以空格分隔。",
	"Global environment":                   "全局环境变量",
	"KEY=value pairs separated by spaces.": "以空格分隔的 KEY=value 对。",
	"Application":                          "应用",
	"Start at login":                       "登录时启动",
	"Show in Dock":                         "在程序坞中显示",
	"Applied when you save.":               "保存后生效。",
	"Language":                             "语言",
	"System":                               "跟随系统",
	"The configuration lives at %s":        "配置文件位于 %s",
	"Saved":                                "已保存",
	"Reverted to the saved configuration":  "已还原为已保存的配置",
	"Login item updated":                   "登录项已更新",
	"Login item: %s":                       "登录项：%s",
	"Save failed: %s":                      "保存失败：%s",
	"Native helper unavailable: %s":        "原生 helper 不可用：%s",

	// Apps
	"Open With applications":     "打开方式应用",
	"Add Application…":           "添加应用…",
	"No applications yet.":       "尚未添加应用。",
	"Remove":                     "移除",
	"Inherit arguments":          "继承参数",
	"Inherit environment":        "继承环境变量",
	"Inherit global arguments":   "继承全局参数",
	"Inherit global environment": "继承全局环境变量",
	"Arguments":                  "参数",
	"Environment":                "环境变量",
	"Choose an Application":      "选择应用",
	"Applications":               "应用程序",
	"Application already added":  "该应用已添加",

	// Actions
	"Finder actions": "Finder 操作",
	"Reset Actions":  "重置操作",

	// New files
	"New File templates":    "新建文件模板",
	"Reset Templates":       "重置模板",
	"Add a template":        "添加模板",
	"Name":                  "名称",
	"Extension":             "扩展名",
	"Add":                   "添加",
	"Enter a template name": "请输入模板名称",

	// Folders
	"Favorite Folders":         "常用文件夹",
	"Add Folder…":              "添加文件夹…",
	"No favorite folders yet.": "尚未添加常用文件夹。",
	"Watched Folders":          "监视的文件夹",
	"Folders where the RClick menu appears. Save, then refresh the Finder menu.": "显示 RClick 菜单的文件夹。保存后请刷新 Finder 菜单。",
	"Choose Favorite Folders": "选择常用文件夹",
	"Choose Watched Folders":  "选择监视的文件夹",
	"Folder already added":    "该文件夹已添加",
	"Folder already watched":  "该文件夹已在监视列表中",

	// Commands
	"Quick Commands":     "快捷命令",
	"Reset Commands":     "重置命令",
	"Add Command":        "添加命令",
	"Ask before running": "运行前确认",
	"Placeholders: {path} {name} {dir} {ext} {name_no_ext} {all}": "占位符：{path} {name} {dir} {ext} {name_no_ext} {all}",
	"New Command":                    "新命令",
	"Run Quick Command?":             "运行快捷命令？",
	"Command completed successfully": "命令执行成功",
	"Command failed":                 "命令执行失败",
}
