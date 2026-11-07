// Package ipc defines the messages exchanged with the Swift FinderSync
// extension and the native helper.
package ipc

import (
	"encoding/json"
	"fmt"
)

const (
	ActionOpen           = "open"
	ActionPerformAction  = "actioning"
	ActionCreateFile     = "Create File"
	ActionOpenCommonDirs = "common-dirs"
	ActionQuickCommand   = "quick-command"
	ActionHeartbeat      = "heartbeat"
	ActionRunning        = "running"
	ActionQuit           = "quit"

	NotificationFromFinder = "RCLICK_FINDER_Main"
	NotificationFromMain   = "RCLICK_MAIN_FINDER"
	NotificationRunning    = "running"
	NotificationQuit       = "quit"
)

type Payload struct {
	Action  string   `json:"action"`
	Target  []string `json:"target"`
	Rid     string   `json:"rid"`
	Trigger string   `json:"trigger"`
}

func New(action string, target []string, rid, trigger string) Payload {
	if target == nil {
		target = []string{}
	}
	return Payload{Action: action, Target: target, Rid: rid, Trigger: trigger}
}

// JSON encodes the payload exactly as Swift's MessagePayload expects it.
func (p Payload) JSON() string {
	if p.Target == nil {
		p.Target = []string{}
	}
	data, err := json.Marshal(p)
	if err != nil {
		return `{"action":"","target":[],"rid":"","trigger":""}`
	}
	return string(data)
}

func Parse(text string) (Payload, error) {
	var p Payload
	if err := json.Unmarshal([]byte(text), &p); err != nil {
		return Payload{}, fmt.Errorf("decode message payload: %w", err)
	}
	if p.Target == nil {
		p.Target = []string{}
	}
	return p, nil
}
