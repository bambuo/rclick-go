package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// Store serializes access to the shared config file. Every process that
// reads the file sees the same revision and keys.
type Store struct {
	mu       sync.RWMutex
	path     string
	current  Config
	onChange func(Config)
}

// Open loads path, creating it with defaults when it does not exist. A
// corrupt file is left untouched and reported; the returned store then
// holds defaults so the menu keeps working while the user fixes the file.
func Open(path string) (*Store, error) {
	if path == "" {
		if override := os.Getenv("RCLICK_CONFIG"); override != "" {
			path = override
		} else {
			path = ConfigFile()
		}
	}
	s := &Store{path: path}
	if err := s.load(); err != nil {
		var notExist *os.PathError
		if errors.As(err, &notExist) && errors.Is(err, os.ErrNotExist) {
			s.current = NewDefault()
			if saveErr := s.persist(s.current); saveErr != nil {
				return s, fmt.Errorf("create default config: %w", saveErr)
			}
			return s, nil
		}
		s.current = NewDefault()
		NormalizeSlices(&s.current)
		return s, err
	}
	return s, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Current() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.current)
}

// SetOnChange registers a callback invoked after every successful save or
// reload, outside the store lock.
func (s *Store) SetOnChange(fn func(Config)) {
	s.mu.Lock()
	s.onChange = fn
	s.mu.Unlock()
}

func (s *Store) Reload() (Config, error) {
	if err := s.load(); err != nil {
		return s.Current(), err
	}
	cfg := s.Current()
	s.notify(cfg)
	return cfg, nil
}

// Save normalizes, validates and atomically replaces the file, bumping
// the revision so the Finder extension can detect the change if it ever
// polls. The saved copy is returned.
func (s *Store) Save(cfg Config) (Config, error) {
	Normalize(&cfg)
	NormalizeSlices(&cfg)
	cfg.SchemaVersion = SchemaVersion
	cfg.Revision = s.currentRevision() + 1
	cfg.UpdatedAt = time.Now().UTC()

	if err := Validate(&cfg); err != nil {
		return s.Current(), err
	}

	s.mu.Lock()
	if err := s.persist(cfg); err != nil {
		s.mu.Unlock()
		return s.Current(), err
	}
	s.current = cloneConfig(cfg)
	s.mu.Unlock()

	s.notify(cfg)
	return cloneConfig(cfg), nil
}

// Update applies fn to a copy of the current config and saves the result.
func (s *Store) Update(fn func(*Config) error) (Config, error) {
	cfg := s.Current()
	if err := fn(&cfg); err != nil {
		return cfg, err
	}
	return s.Save(cfg)
}

func (s *Store) currentRevision() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current.Revision
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	cfg, err := decode(data)
	if err != nil {
		return fmt.Errorf("decode %s: %w", s.path, err)
	}
	if err := Validate(&cfg); err != nil {
		return fmt.Errorf("%s: %w", s.path, err)
	}

	s.mu.Lock()
	s.current = cfg
	s.mu.Unlock()
	return nil
}

// persist expects the caller to hold the write lock when one is needed.
func (s *Store) persist(cfg Config) error {
	if err := EnsureRoot(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	return writeFileAtomic(s.path, data)
}

func (s *Store) notify(cfg Config) {
	s.mu.RLock()
	fn := s.onChange
	s.mu.RUnlock()
	if fn != nil {
		fn(cloneConfig(cfg))
	}
}

func decode(data []byte) (Config, error) {
	cfg := Config{
		SchemaVersion: SchemaVersion,
		Menu: MenuSettings{
			ShowForItems:     true,
			ShowForContainer: true,
			ShowForSidebar:   true,
			ShowToolbarItem:  true,
		},
		Defaults: DefaultSettings{NewFileName: "Untitled"},
		UI:       UISettings{ShowMenuBarExtra: true},
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = SchemaVersion
	}
	NormalizeSlices(&cfg)
	return cfg, nil
}

func cloneConfig(cfg Config) Config {
	out := cfg
	out.Apps = append([]OpenWithApp(nil), cfg.Apps...)
	for i := range out.Apps {
		out.Apps[i].Arguments = append([]string(nil), cfg.Apps[i].Arguments...)
		out.Apps[i].Environment = cloneMap(cfg.Apps[i].Environment)
	}
	out.Actions = append([]Action(nil), cfg.Actions...)
	out.NewFiles = append([]NewFile(nil), cfg.NewFiles...)
	out.CommonDirs = append([]CommonDir(nil), cfg.CommonDirs...)
	out.PermissionDirs = append([]PermissionDir(nil), cfg.PermissionDirs...)
	out.QuickCommands = append([]QuickCommand(nil), cfg.QuickCommands...)
	return out
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
