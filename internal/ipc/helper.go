package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Event is one line the helper writes to stdout.
type Event struct {
	Type      string
	Name      string
	Payload   string
	Message   string
	ID        int64
	Confirmed bool
}

var ErrHelperNotRunning = errors.New("native helper is not running")

// Helper owns the long-lived Swift helper process. The helper handles the
// macOS APIs mygo does not expose: DistributedNotificationCenter,
// AirDrop, open-with arguments/environment, Trash and the hidden flag.
type Helper struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	pending map[int64]chan bool
	nextID  int64
	done    chan struct{}
	closed  bool

	onEvent func(Event)
}

// FindHelper looks for RClickHelper in the usual development and bundle
// locations.
func FindHelper() string {
	if path := os.Getenv("RCLICK_HELPER"); path != "" {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "RClickHelper")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	for _, candidate := range []string{
		filepath.Join("native", "helper", "bin", "RClickHelper"),
		"RClickHelper",
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}
	return ""
}

func StartHelper(onEvent func(Event)) (*Helper, error) {
	path := FindHelper()
	if path == "" {
		return nil, errors.New("RClickHelper not found; build it with `make helper`")
	}

	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", path, err)
	}

	h := &Helper{
		cmd:     cmd,
		stdin:   stdin,
		pending: make(map[int64]chan bool),
		done:    make(chan struct{}),
		onEvent: onEvent,
	}

	go h.readEvents(stdout)
	go h.logErrors(stderr)
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("RClickHelper exited: %v", err)
		}
		h.finish()
	}()

	return h, nil
}

func (h *Helper) Send(name string, payload Payload) error {
	return h.write(map[string]any{"cmd": "send", "name": name, "payload": payload.JSON()})
}

func (h *Helper) SendRaw(name, payload string) error {
	return h.write(map[string]any{"cmd": "send", "name": name, "payload": payload})
}

func (h *Helper) SetClipboard(text string) error {
	return h.write(map[string]any{"cmd": "clipboard", "text": text})
}

func (h *Helper) Open(paths []string) error {
	return h.write(map[string]any{"cmd": "open", "paths": paths})
}

func (h *Helper) OpenWith(appPath string, paths, arguments []string, environment map[string]string) error {
	if environment == nil {
		environment = map[string]string{}
	}
	return h.write(map[string]any{
		"cmd":         "openWith",
		"appPath":     appPath,
		"paths":       paths,
		"arguments":   arguments,
		"environment": environment,
	})
}

func (h *Helper) AirDrop(paths []string) error {
	return h.write(map[string]any{"cmd": "airdrop", "paths": paths})
}

func (h *Helper) Trash(paths []string) error {
	return h.write(map[string]any{"cmd": "trash", "paths": paths})
}

func (h *Helper) SetHidden(paths []string, hidden bool) error {
	return h.write(map[string]any{"cmd": "hidden", "paths": paths, "hidden": hidden})
}

// Confirm shows a native alert through the helper and waits for the user.
func (h *Helper) Confirm(ctx context.Context, title, message string, dangerous bool) (bool, error) {
	h.mu.Lock()
	if h.closed || h.stdin == nil {
		h.mu.Unlock()
		return false, ErrHelperNotRunning
	}
	h.nextID++
	id := h.nextID
	result := make(chan bool, 1)
	h.pending[id] = result
	err := h.writeLocked(map[string]any{
		"cmd":       "confirm",
		"id":        id,
		"title":     title,
		"message":   message,
		"dangerous": dangerous,
	})
	h.mu.Unlock()

	if err != nil {
		h.dropPending(id)
		return false, err
	}

	select {
	case confirmed := <-result:
		return confirmed, nil
	case <-ctx.Done():
		h.dropPending(id)
		return false, ctx.Err()
	case <-h.done:
		return false, ErrHelperNotRunning
	}
}

func (h *Helper) Quit() {
	_ = h.write(map[string]any{"cmd": "quit"})
}

// Close asks the helper to quit and reaps it, killing it after a grace
// period.
func (h *Helper) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	stdin := h.stdin
	h.stdin = nil
	h.mu.Unlock()

	if stdin != nil {
		_ = writeJSON(stdin, map[string]any{"cmd": "quit"})
		_ = stdin.Close()
	}

	select {
	case <-h.done:
	case <-time.After(2 * time.Second):
		if h.cmd != nil && h.cmd.Process != nil {
			_ = h.cmd.Process.Kill()
		}
		select {
		case <-h.done:
		case <-time.After(time.Second):
		}
	}
}

func (h *Helper) write(command map[string]any) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.stdin == nil {
		return ErrHelperNotRunning
	}
	return h.writeLocked(command)
}

func (h *Helper) writeLocked(command map[string]any) error {
	if h.stdin == nil {
		return ErrHelperNotRunning
	}
	return writeJSON(h.stdin, command)
}

func writeJSON(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

func (h *Helper) readEvents(stdout io.Reader) {
	reader := bufio.NewReaderSize(stdout, 64*1024)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			h.handleLine(line)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("read helper output: %v", err)
			}
			return
		}
	}
}

func (h *Helper) logErrors(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			log.Printf("RClickHelper: %s", line)
		}
	}
}

type wireEvent struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Payload   json.RawMessage `json:"payload"`
	Message   string          `json:"message"`
	ID        int64           `json:"id"`
	Confirmed bool            `json:"confirmed"`
}

func (h *Helper) handleLine(line string) {
	var wire wireEvent
	if err := json.Unmarshal([]byte(line), &wire); err != nil {
		log.Printf("helper emitted invalid JSON: %s", line)
		return
	}

	switch wire.Type {
	case "confirmResult":
		h.completePending(wire.ID, wire.Confirmed)
		return
	case "message", "error", "ready":
		if h.onEvent != nil {
			h.onEvent(Event{
				Type:    wire.Type,
				Name:    wire.Name,
				Payload: decodePayload(wire.Payload),
				Message: wire.Message,
				ID:      wire.ID,
			})
		}
	default:
		log.Printf("helper event: %s", line)
	}
}

func decodePayload(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	if raw[0] == '"' {
		var text string
		if err := json.Unmarshal(raw, &text); err == nil {
			return text
		}
	}
	return string(raw)
}

func (h *Helper) completePending(id int64, confirmed bool) {
	h.mu.Lock()
	ch, ok := h.pending[id]
	if ok {
		delete(h.pending, id)
	}
	h.mu.Unlock()
	if ok {
		ch <- confirmed
		close(ch)
	}
}

func (h *Helper) dropPending(id int64) {
	h.mu.Lock()
	delete(h.pending, id)
	h.mu.Unlock()
}

func (h *Helper) finish() {
	h.mu.Lock()
	if !h.closed {
		h.closed = true
	}
	h.stdin = nil
	pending := h.pending
	h.pending = make(map[int64]chan bool)
	select {
	case <-h.done:
	default:
		close(h.done)
	}
	h.mu.Unlock()

	for _, ch := range pending {
		ch <- false
		close(ch)
	}
}
