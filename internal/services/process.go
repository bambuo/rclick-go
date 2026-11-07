package services

import (
	"bytes"
	"context"
	"os/exec"
	"unicode/utf8"
)

const maxCommandOutput = 8 * 1024

type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

func (r CommandResult) Succeeded() bool { return r.Err == nil && r.ExitCode == 0 }

// RunShell executes script with the user's login shell, so aliases and
// PATH additions behave as they do in Terminal.
func RunShell(ctx context.Context, script string) CommandResult {
	cmd := exec.CommandContext(ctx, "/bin/zsh", "-lc", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := CommandResult{
		Stdout: limitOutput(stdout.String()),
		Stderr: limitOutput(stderr.String()),
		Err:    err,
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	} else if err != nil {
		result.ExitCode = 1
	}
	return result
}

func limitOutput(text string) string {
	if len(text) <= maxCommandOutput {
		return text
	}
	cut := maxCommandOutput
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + "\n[truncated]"
}
