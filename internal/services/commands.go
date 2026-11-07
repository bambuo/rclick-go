package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bambuo/rclick-go/internal/config"
	"github.com/bambuo/rclick-go/internal/i18n"
)

type CommandRunner struct {
	Store   *config.Store
	Runner  func(ctx context.Context, script string) CommandResult
	Confirm func(ctx context.Context, title, message string, dangerous bool) (bool, error)
	Notify  func(title, body string)

	ConfirmInterval time.Duration

	mu            sync.Mutex
	lastConfirmed time.Time
}

func (r *CommandRunner) Run(ctx context.Context, commandID string, targets []string) (CommandResult, error) {
	cfg := r.Store.Current()
	var command *config.QuickCommand
	for i := range cfg.QuickCommands {
		if cfg.QuickCommands[i].ID == commandID && cfg.QuickCommands[i].Enabled {
			command = &cfg.QuickCommands[i]
			break
		}
	}
	if command == nil {
		return CommandResult{}, fmt.Errorf("quick command %q not found or disabled", commandID)
	}
	if len(targets) == 0 {
		return CommandResult{}, fmt.Errorf("quick command %q has no targets", commandID)
	}

	resolved := ResolveTemplate(command.Template, targets)
	displayName := i18n.CommandName(command.ID, command.Name)
	if r.needsConfirmation(*command) {
		if r.Confirm == nil {
			return CommandResult{}, fmt.Errorf("confirmation unavailable")
		}
		confirmed, err := r.Confirm(ctx, i18n.T("Run Quick Command?"), displayName+"\n\n"+resolved, command.Dangerous)
		if err != nil {
			return CommandResult{}, err
		}
		if !confirmed {
			return CommandResult{}, fmt.Errorf("cancelled")
		}
		r.markConfirmed()
	}

	runner := r.Runner
	if runner == nil {
		runner = RunShell
	}
	result := runner(ctx, resolved)
	r.notify(displayName, result)
	return result, nil
}

func (r *CommandRunner) needsConfirmation(command config.QuickCommand) bool {
	if !command.Dangerous {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lastConfirmed.IsZero() {
		return true
	}
	interval := r.ConfirmInterval
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return time.Since(r.lastConfirmed) > interval
}

func (r *CommandRunner) markConfirmed() {
	r.mu.Lock()
	r.lastConfirmed = time.Now()
	r.mu.Unlock()
}

func (r *CommandRunner) notify(name string, result CommandResult) {
	body := strings.TrimSpace(result.Stderr)
	if body == "" {
		body = strings.TrimSpace(result.Stdout)
	}
	body = truncateRunes(body, 200)
	if body == "" {
		if result.Succeeded() {
			body = i18n.T("Command completed successfully")
		} else {
			body = i18n.T("Command failed")
		}
	}

	title := "✅ " + name
	if !result.Succeeded() {
		title = "❌ " + name
	}
	if result.Err != nil {
		log.Printf("quick command %s: %v", name, result.Err)
	} else {
		log.Printf("quick command %s exited %d", name, result.ExitCode)
	}
	if r.Notify != nil {
		r.Notify(title, body)
	}
}

func truncateRunes(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit])
}
