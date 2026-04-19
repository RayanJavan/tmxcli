package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Store struct {
	path string
	mu   sync.Mutex
}

type Snapshot struct {
	Processes []*LoggingProcess `json:"processes"`
}

type LoggingProcess struct {
	ID       int                        `json:"id"`
	LogPath  string                     `json:"log_path"`
	Created  time.Time                  `json:"created"`
	Stopped  *time.Time                 `json:"stopped,omitempty"`
	Active   bool                       `json:"active"`
	Sessions map[string]AttachedSession `json:"sessions,omitempty"`
}

type AttachedSession struct {
	Name     string    `json:"name"`
	Target   string    `json:"target"`
	Attached time.Time `json:"attached"`
}

func New(path string) *Store { return &Store{path: path} }

func (s *Store) Load() (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked()
}

func (s *Store) Update(fn func(*Snapshot) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	if err := fn(snap); err != nil {
		return err
	}
	sort.Slice(snap.Processes, func(i, j int) bool { return snap.Processes[i].ID < snap.Processes[j].ID })
	return s.saveUnlocked(snap)
}

func (s *Store) loadUnlocked() (*Snapshot, error) {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return &Snapshot{Processes: []*LoggingProcess{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	if snap.Processes == nil {
		snap.Processes = []*LoggingProcess{}
	}
	return &snap, nil
}

func (s *Store) saveUnlocked(snap *Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
