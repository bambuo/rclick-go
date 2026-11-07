package app

import (
	"context"
	"log"
	"time"

	"github.com/bambuo/rclick-go/internal/ipc"
)

func (a *App) handleMessage(text string) {
	payload, err := ipc.Parse(text)
	if err != nil {
		log.Printf("finder message: %v", err)
		return
	}

	switch payload.Action {
	case ipc.ActionHeartbeat:
		a.replyHeartbeat()
	case ipc.ActionOpen:
		if a.launcher == nil {
			return
		}
		if err := a.launcher.Open(payload.Rid, payload.Target); err != nil {
			log.Printf("open with %s: %v", payload.Rid, err)
		}
	case ipc.ActionPerformAction:
		a.performAction(payload)
	case ipc.ActionCreateFile:
		if a.files == nil || len(payload.Target) == 0 {
			return
		}
		if err := a.files.CreateFile(payload.Target[0], payload.Rid); err != nil {
			log.Printf("create file: %v", err)
		}
	case ipc.ActionOpenCommonDirs:
		if a.files == nil {
			return
		}
		if err := a.files.Open(payload.Target); err != nil {
			log.Printf("open common dirs: %v", err)
		}
	case ipc.ActionQuickCommand:
		if a.commands == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if _, err := a.commands.Run(ctx, payload.Rid, payload.Target); err != nil {
			log.Printf("quick command %s: %v", payload.Rid, err)
		}
	case ipc.ActionRunning, ipc.ActionQuit:
		// The extension's echo of our own state; nothing to do.
	default:
		log.Printf("unknown finder action %q", payload.Action)
	}
}

func (a *App) performAction(payload ipc.Payload) {
	if a.files == nil {
		return
	}
	cfg := a.store.Current()
	var actionID string
	for _, action := range cfg.Actions {
		if action.ID == payload.Rid && action.Enabled {
			actionID = action.ID
			break
		}
	}
	if actionID == "" {
		log.Printf("action %q not found or disabled", payload.Rid)
		return
	}

	var err error
	switch actionID {
	case "copy-path":
		err = a.files.CopyPaths(payload.Target)
	case "delete-direct":
		err = a.files.Delete(payload.Target, payload.Trigger)
	case "hide":
		err = a.files.SetHidden(payload.Target, true, payload.Trigger)
	case "unhide":
		err = a.files.SetHidden(payload.Target, false, payload.Trigger)
	case "airdrop":
		err = a.files.AirDrop(payload.Target, payload.Trigger)
	default:
		log.Printf("unhandled action %q", actionID)
		return
	}
	if err != nil {
		log.Printf("action %s: %v", actionID, err)
	}
}
