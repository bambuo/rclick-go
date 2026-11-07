// Package app wires the configuration, the Finder IPC and the mygo UI.
package app

import (
	"context"
	_ "embed"
	"log"
	"os"
	"sync"
	"time"

	"github.com/bambuo/rclick-go/internal/config"
	"github.com/bambuo/rclick-go/internal/i18n"
	"github.com/bambuo/rclick-go/internal/ipc"
	"github.com/bambuo/rclick-go/internal/services"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

const Version = "3.0.0"

//go:embed assets/tray.png
var trayIcon []byte

type App struct {
	store    *config.Store
	helper   *ipc.Helper
	files    *services.FileOperations
	launcher *services.AppLauncher
	commands *services.CommandRunner

	tray   *mygo.Tray
	window *mygo.Window

	quitting bool
	settings settingsState

	heartbeatMu   sync.Mutex
	lastHeartbeat time.Time
}

func New(store *config.Store) *App {
	return &App{store: store}
}

func (a *App) Run() error {
	i18n.Apply(a.store.Current().UI.Language)
	mygo.App.SetName("RClick")
	mygo.App.SetVersion(Version)
	mygo.App.SetActivationPolicy(activationPolicy(a.store.Current().UI.ShowInDock))

	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}

	mygo.App.WhenReady(a.ready)
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.quitting = true })
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { a.shutdown() })
	return mygo.App.Run()
}

func (a *App) ready() {
	helper, err := ipc.StartHelper(a.handleHelperEvent)
	if err != nil {
		log.Printf("native helper unavailable: %v", err)
		a.settings.status = i18n.Sprintf("Native helper unavailable: %s", err.Error())
	} else {
		a.helper = helper
	}

	a.files = &services.FileOperations{
		Store:     a.store,
		Helper:    a.helper,
		Clipboard: func(text string) error { return mygo.Clipboard.Write(transfer.TextData(text)) },
	}
	a.launcher = &services.AppLauncher{Store: a.store, Helper: a.helper}
	a.commands = &services.CommandRunner{
		Store:   a.store,
		Confirm: a.confirm,
		Notify:  a.notify,
	}

	a.createTray()
	a.createSettingsWindow()
	a.refreshSettings()
	a.sendRunning()

	if os.Getenv("RCLICK_OPEN_SETTINGS") == "1" {
		a.showSettings()
		mygo.App.Focus()
	}
}

func (a *App) shutdown() {
	if a.helper != nil {
		_ = a.helper.Send(ipc.NotificationQuit, ipc.New(ipc.ActionQuit, nil, "", ""))
		a.helper.Close()
		a.helper = nil
	}
	if a.tray != nil {
		a.tray.Destroy()
		a.tray = nil
	}
}

func (a *App) handleHelperEvent(event ipc.Event) {
	switch event.Type {
	case "message":
		go a.handleMessage(event.Payload)
	case "error":
		log.Printf("native helper: %s", event.Message)
	case "ready":
		log.Printf("native helper ready")
	}
}

func (a *App) sendRunning() {
	if a.helper == nil {
		return
	}
	cfg := a.store.Current()
	targets := make([]string, 0, len(cfg.PermissionDirs))
	for _, dir := range cfg.PermissionDirs {
		if dir.URL != "" {
			targets = append(targets, dir.URL)
		}
	}
	if err := a.helper.Send(ipc.NotificationRunning, ipc.New(ipc.ActionRunning, targets, "", "")); err != nil {
		log.Printf("send running: %v", err)
	}
}

// replyHeartbeat answers a heartbeat, but not more than once every two
// seconds: the extension heartbeats when it receives running, so answering
// every heartbeat would make the two processes ping-pong forever.
func (a *App) replyHeartbeat() {
	a.heartbeatMu.Lock()
	if time.Since(a.lastHeartbeat) < 2*time.Second {
		a.heartbeatMu.Unlock()
		return
	}
	a.lastHeartbeat = time.Now()
	a.heartbeatMu.Unlock()
	a.sendRunning()
}

func (a *App) confirm(ctx context.Context, title, message string, dangerous bool) (bool, error) {
	if a.helper == nil {
		return false, ipc.ErrHelperNotRunning
	}
	return a.helper.Confirm(ctx, title, message, dangerous)
}

func (a *App) notify(title, body string) {
	if !mygo.NotificationsSupported() {
		log.Printf("%s: %s", title, body)
		return
	}
	notification := mygo.NewNotification(mygo.NotificationOptions{
		ID:    notificationID(),
		Title: title,
		Body:  body,
	})
	if err := notification.Show(); err != nil {
		log.Printf("notification: %v", err)
	}
}

func notificationID() string {
	return "rclick-" + time.Now().Format("20060102150405.000000000")
}

func activationPolicy(showInDock bool) mygo.ActivationPolicy {
	if showInDock {
		return mygo.ActivationPolicyRegular
	}
	return mygo.ActivationPolicyAccessory
}
