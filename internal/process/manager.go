package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/example/tmxlog/internal/state"
	"github.com/example/tmxlog/internal/tmux"
)

type Manager struct {
	store *state.Store
	tmux  *tmux.Client
}

func New(store *state.Store, tmux *tmux.Client) *Manager {
	return &Manager{store: store, tmux: tmux}
}

func (m *Manager) Start(logPath string) (*state.LoggingProcess, error) {
	if logPath == "" {
		return nil, fmt.Errorf("log path is required")
	}
	var created *state.LoggingProcess
	err := m.store.Update(func(s *state.Snapshot) error {
		now := time.Now().UTC()
		id := nextID(s.Processes)
		p := &state.LoggingProcess{ID: id, LogPath: logPath, Created: now, Active: true, Sessions: map[string]state.AttachedSession{}}
		s.Processes = append(s.Processes, p)
		created = p
		return nil
	})
	return created, err
}

func nextID(ps []*state.LoggingProcess) int {
	if len(ps) == 0 {
		return 0
	}
	seen := map[int]bool{}
	for _, p := range ps {
		seen[p.ID] = true
	}
	for i := 0; ; i++ {
		if !seen[i] {
			return i
		}
	}
}

func (m *Manager) Stop(processID *int) (*state.LoggingProcess, error) {
	var stopped *state.LoggingProcess
	err := m.store.Update(func(s *state.Snapshot) error {
		p, err := selectProcess(s.Processes, processID, true)
		if err != nil {
			return err
		}
		for name := range p.Sessions {
			if derr := m.detachSession(p.ID, name); derr != nil {
				return derr
			}
		}
		now := time.Now().UTC()
		p.Active = false
		p.Stopped = &now
		stopped = p
		return nil
	})
	return stopped, err
}

func (m *Manager) Attach(processID *int, target string) (*state.LoggingProcess, string, error) {
	resolved, err := m.ResolveSession(target)
	if err != nil {
		return nil, "", err
	}
	var updated *state.LoggingProcess
	err = m.store.Update(func(s *state.Snapshot) error {
		p, err := selectProcess(s.Processes, processID, true)
		if err != nil {
			return err
		}
		if p.Sessions == nil {
			p.Sessions = map[string]state.AttachedSession{}
		}
		if _, ok := p.Sessions[resolved]; ok {
			updated = p
			return nil
		}
		if err := m.attachSessionPanes(p.ID, resolved); err != nil {
			return err
		}
		if err := m.installHooks(p.ID, resolved); err != nil {
			return err
		}
		p.Sessions[resolved] = state.AttachedSession{Name: resolved, Target: target, Attached: time.Now().UTC()}
		updated = p
		return nil
	})
	return updated, resolved, err
}

func (m *Manager) Detach(processID *int, target string) (*state.LoggingProcess, string, error) {
	resolved, err := m.ResolveSession(target)
	if err != nil {
		return nil, "", err
	}
	var updated *state.LoggingProcess
	err = m.store.Update(func(s *state.Snapshot) error {
		p, err := selectProcess(s.Processes, processID, true)
		if err != nil {
			return err
		}
		if _, ok := p.Sessions[resolved]; !ok {
			updated = p
			return nil
		}
		if err := m.detachSession(p.ID, resolved); err != nil {
			return err
		}
		delete(p.Sessions, resolved)
		updated = p
		return nil
	})
	return updated, resolved, err
}

func (m *Manager) AttachSessionPanes(processID int, session string) error {
	return m.attachSessionPanes(processID, session)
}

func (m *Manager) ResolveSession(target string) (string, error) {
	if target == "" {
		if os.Getenv("TMUX") == "" {
			return "", fmt.Errorf("no session target specified and not running inside tmux")
		}
		target = "="
	}
	return m.tmux.Output("display-message", "-p", "-t", target, "#S")
}

func (m *Manager) attachSessionPanes(processID int, session string) error {
	panes, err := m.tmux.Output("list-panes", "-t", session, "-F", "#{pane_id}")
	if err != nil {
		return err
	}
	if panes == "" {
		return nil
	}
	for _, pane := range strings.Split(panes, "\n") {
		cmd := fmt.Sprintf("tmxlog _ingest --process %d --session %s --pane %s", processID, shellQuote(session), shellQuote(pane))
		if err := m.tmux.Run("pipe-pane", "-o", "-t", pane, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) installHooks(processID int, session string) error {
	cmd := fmt.Sprintf("tmxlog _attach-session-panes --process %d --session %s", processID, shellQuote(session))
	if err := m.tmux.Run("set-hook", "-t", session, fmt.Sprintf("tmxlog-new-window-%d", processID), "after-new-window", cmd); err != nil {
		return err
	}
	if err := m.tmux.Run("set-hook", "-t", session, fmt.Sprintf("tmxlog-split-window-%d", processID), "after-split-window", cmd); err != nil {
		return err
	}
	return nil
}

func (m *Manager) detachSession(processID int, session string) error {
	panes, err := m.tmux.Output("list-panes", "-t", session, "-F", "#{pane_id}")
	if err == nil && panes != "" {
		for _, pane := range strings.Split(panes, "\n") {
			_ = m.tmux.Run("pipe-pane", "-t", pane)
		}
	}
	_ = m.tmux.Run("set-hook", "-u", "-t", session, fmt.Sprintf("tmxlog-new-window-%d", processID))
	_ = m.tmux.Run("set-hook", "-u", "-t", session, fmt.Sprintf("tmxlog-split-window-%d", processID))
	return nil
}

func selectProcess(ps []*state.LoggingProcess, requested *int, activeOnly bool) (*state.LoggingProcess, error) {
	if len(ps) == 0 {
		return nil, fmt.Errorf("no logging processes found")
	}
	if requested != nil {
		for _, p := range ps {
			if p.ID == *requested {
				if activeOnly && !p.Active {
					return nil, fmt.Errorf("process %d is not active", p.ID)
				}
				return p, nil
			}
		}
		return nil, fmt.Errorf("process %d not found", *requested)
	}
	sorted := append([]*state.LoggingProcess(nil), ps...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Created.After(sorted[j].Created) })
	for _, p := range sorted {
		if !activeOnly || p.Active {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no active logging process found")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func (m *Manager) Ingest(processID int, session, pane string, in io.Reader) error {
	snap, err := m.store.Load()
	if err != nil {
		return err
	}
	var proc *state.LoggingProcess
	for _, p := range snap.Processes {
		if p.ID == processID {
			proc = p
			break
		}
	}
	if proc == nil {
		return fmt.Errorf("process %d not found", processID)
	}
	if !proc.Active {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(proc.LogPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(proc.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	lock, err := os.OpenFile(proc.LogPath+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer lock.Close()

	r := bufio.NewReader(in)
	for {
		line, err := r.ReadString('\n')
		if line != "" {
			if lerr := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); lerr != nil {
				return lerr
			}
			_, werr := fmt.Fprintf(f, "%s [proc:%d] [session:%s] [pane:%s] %s", time.Now().UTC().Format(time.RFC3339Nano), processID, session, pane, line)
			uerr := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
			if werr != nil {
				return werr
			}
			if uerr != nil {
				return uerr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
