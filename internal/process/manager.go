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
	self  string
}

func New(store *state.Store, tmux *tmux.Client) *Manager {
	self := "tmxlog"
	if exe, err := os.Executable(); err == nil && exe != "" {
		self = exe
	}
	return &Manager{store: store, tmux: tmux, self: self}
}

func (m *Manager) Start(logPath string) (*state.LoggingProcess, error) {
	if logPath == "" {
		return nil, fmt.Errorf("log path is required")
	}
	if err := ensureLogFile(logPath); err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
	}
	if err := appendLifecycleLine(logPath, created.ID, "started"); err != nil {
		return nil, err
	}
	return created, nil
}

func nextID(ps []*state.LoggingProcess) int {
	if len(ps) == 0 {
		return 0
	}
	seen := map[int]bool{}
	for _, p := range ps {
		if p.Active {
			seen[p.ID] = true
		}
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
		p, idx, err := selectProcessWithIndex(s.Processes, processID, true)
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
		// Remove from queue when stopped so indexing and queue semantics mirror tmux-like active objects.
		s.Processes = append(s.Processes[:idx], s.Processes[idx+1:]...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := appendLifecycleLine(stopped.LogPath, stopped.ID, "stopped"); err != nil {
		return nil, err
	}
	return stopped, nil
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
		cmd := fmt.Sprintf("%s _ingest --process %d --session %s --pane %s", shellQuote(m.self), processID, shellQuote(session), shellQuote(pane))
		if err := m.tmux.Run("pipe-pane", "-o", "-t", pane, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) installHooks(processID int, session string) error {
	inner := fmt.Sprintf("%s _attach-session-panes --process %d --session %s", shellQuote(m.self), processID, shellQuote(session))
	cmd := fmt.Sprintf("run-shell %s", shellQuote(inner))
	if err := m.tmux.Run("set-hook", "-t", session, "after-new-window", cmd); err != nil {
		return err
	}
	if err := m.tmux.Run("set-hook", "-t", session, "after-split-window", cmd); err != nil {
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
	_ = m.tmux.Run("set-hook", "-u", "-t", session, "after-new-window")
	_ = m.tmux.Run("set-hook", "-u", "-t", session, "after-split-window")
	return nil
}

func selectProcessWithIndex(ps []*state.LoggingProcess, requested *int, activeOnly bool) (*state.LoggingProcess, int, error) {
	if len(ps) == 0 {
		return nil, -1, fmt.Errorf("no logging processes found")
	}
	if requested != nil {
		for i, p := range ps {
			if p.ID == *requested {
				if activeOnly && !p.Active {
					return nil, -1, fmt.Errorf("process %d is not active", p.ID)
				}
				return p, i, nil
			}
		}
		return nil, -1, fmt.Errorf("process %d not found", *requested)
	}
	sorted := make([]struct {
		idx int
		p   *state.LoggingProcess
	}, 0, len(ps))
	for i, p := range ps {
		sorted = append(sorted, struct {
			idx int
			p   *state.LoggingProcess
		}{idx: i, p: p})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].p.Created.After(sorted[j].p.Created) })
	for _, item := range sorted {
		if !activeOnly || item.p.Active {
			return item.p, item.idx, nil
		}
	}
	return nil, -1, fmt.Errorf("no active logging process found")
}

func selectProcess(ps []*state.LoggingProcess, requested *int, activeOnly bool) (*state.LoggingProcess, error) {
	p, _, err := selectProcessWithIndex(ps, requested, activeOnly)
	return p, err
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
	if err := ensureLogFile(proc.LogPath); err != nil {
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

func ensureLogFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

func appendLifecycleLine(path string, processID int, action string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s [proc:%d] [system] %s logging process\n", time.Now().UTC().Format(time.RFC3339Nano), processID, action)
	return err
}
