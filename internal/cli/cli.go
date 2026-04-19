package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
)

type Command struct {
	Name string
	Args []string
}

type StartOpts struct{ LogPath string }
type ProcOpts struct {
	ProcessID *int
	Target    string
}
type IngestOpts struct {
	ProcessID     int
	Session, Pane string
}

type AttachSessionPanesOpts struct {
	ProcessID int
	Session   string
}

func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{}, errors.New("usage: tmxlog <start|stop|attach|detach|_ingest|_attach-session-panes>")
	}
	return Command{Name: args[0], Args: args[1:]}, nil
}

func ParseStart(args []string) (StartOpts, error) {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return StartOpts{}, err
	}
	if fs.NArg() != 1 {
		return StartOpts{}, fmt.Errorf("usage: tmxlog start <log_path>")
	}
	return StartOpts{LogPath: fs.Arg(0)}, nil
}

func ParseProcAndTarget(name string, args []string) (ProcOpts, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pid := fs.String("process", "", "logging process id")
	fs.StringVar(pid, "p", "", "logging process id")
	target := fs.String("target", "", "tmux target session")
	fs.StringVar(target, "t", "", "tmux target session")
	if err := fs.Parse(args); err != nil {
		return ProcOpts{}, err
	}
	var p *int
	if *pid != "" {
		v, err := strconv.Atoi(*pid)
		if err != nil {
			return ProcOpts{}, fmt.Errorf("invalid process id %q", *pid)
		}
		p = &v
	}
	return ProcOpts{ProcessID: p, Target: *target}, nil
}

func ParseProc(name string, args []string) (ProcOpts, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pid := fs.String("process", "", "logging process id")
	fs.StringVar(pid, "p", "", "logging process id")
	if err := fs.Parse(args); err != nil {
		return ProcOpts{}, err
	}
	var p *int
	if *pid != "" {
		v, err := strconv.Atoi(*pid)
		if err != nil {
			return ProcOpts{}, fmt.Errorf("invalid process id %q", *pid)
		}
		p = &v
	}
	return ProcOpts{ProcessID: p}, nil
}

func ParseIngest(args []string) (IngestOpts, error) {
	fs := flag.NewFlagSet("_ingest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pid := fs.Int("process", -1, "logging process id")
	session := fs.String("session", "", "session name")
	pane := fs.String("pane", "", "pane id")
	if err := fs.Parse(args); err != nil {
		return IngestOpts{}, err
	}
	if *pid < 0 || *session == "" || *pane == "" {
		return IngestOpts{}, fmt.Errorf("usage: tmxlog _ingest --process <id> --session <name> --pane <id>")
	}
	return IngestOpts{ProcessID: *pid, Session: *session, Pane: *pane}, nil
}

func ParseAttachSessionPanes(args []string) (AttachSessionPanesOpts, error) {
	fs := flag.NewFlagSet("_attach-session-panes", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pid := fs.Int("process", -1, "logging process id")
	session := fs.String("session", "", "session name")
	if err := fs.Parse(args); err != nil {
		return AttachSessionPanesOpts{}, err
	}
	if *pid < 0 || *session == "" {
		return AttachSessionPanesOpts{}, fmt.Errorf("usage: tmxlog _attach-session-panes --process <id> --session <name>")
	}
	return AttachSessionPanesOpts{ProcessID: *pid, Session: *session}, nil
}
