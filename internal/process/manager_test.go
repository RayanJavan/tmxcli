package process

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/tmxlog/internal/state"
	"github.com/example/tmxlog/internal/tmux"
)

func TestNextID(t *testing.T) {
	ps := []*state.LoggingProcess{{ID: 0, Active: true}, {ID: 2, Active: true}, {ID: 3, Active: true}}
	if got := nextID(ps); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

func TestNextIDIgnoresInactive(t *testing.T) {
	ps := []*state.LoggingProcess{{ID: 0, Active: false}, {ID: 1, Active: false}}
	if got := nextID(ps); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestSelectProcessMostRecentActive(t *testing.T) {
	now := time.Now().UTC()
	ps := []*state.LoggingProcess{
		{ID: 0, Created: now.Add(-2 * time.Hour), Active: true},
		{ID: 1, Created: now.Add(-1 * time.Hour), Active: false},
		{ID: 2, Created: now, Active: true},
	}
	p, err := selectProcess(ps, nil, true)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.ID != 2 {
		t.Fatalf("expected ID 2 got %d", p.ID)
	}
}

func TestSelectProcessByID(t *testing.T) {
	ps := []*state.LoggingProcess{{ID: 4, Active: true}}
	id := 4
	p, err := selectProcess(ps, &id, true)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.ID != 4 {
		t.Fatalf("expected ID 4 got %d", p.ID)
	}
}

func TestStartCreatesLogAndStopRemovesFromQueue(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	logPath := filepath.Join(tmp, "test.log")
	mgr := New(state.New(statePath), tmux.New())

	proc, err := mgr.Start(logPath)
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if proc.ID != 0 {
		t.Fatalf("expected process id 0 got %d", proc.ID)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}

	stopped, err := mgr.Stop(&proc.ID)
	if err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if stopped.ID != 0 {
		t.Fatalf("expected stopped id 0 got %d", stopped.ID)
	}

	next, err := mgr.Start(filepath.Join(tmp, "next.log"))
	if err != nil {
		t.Fatalf("second start failed: %v", err)
	}
	if next.ID != 0 {
		t.Fatalf("expected queue to reuse id 0, got %d", next.ID)
	}
}
