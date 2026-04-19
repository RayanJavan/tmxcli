package process

import (
	"testing"
	"time"

	"github.com/example/tmxlog/internal/state"
)

func TestNextID(t *testing.T) {
	ps := []*state.LoggingProcess{{ID: 0}, {ID: 2}, {ID: 3}}
	if got := nextID(ps); got != 1 {
		t.Fatalf("expected 1, got %d", got)
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
