package app

import (
	"fmt"
	"io"

	"github.com/example/tmxlog/internal/cli"
	"github.com/example/tmxlog/internal/process"
	"github.com/example/tmxlog/internal/state"
	"github.com/example/tmxlog/internal/tmux"
	"github.com/example/tmxlog/internal/util"
)

func Run(args []string, stdin io.Reader, stdout, _ io.Writer) error {
	cmd, err := cli.Parse(args)
	if err != nil {
		return err
	}
	statePath, err := util.StateFilePath()
	if err != nil {
		return err
	}
	mgr := process.New(state.New(statePath), tmux.New())

	switch cmd.Name {
	case "start":
		opts, err := cli.ParseStart(cmd.Args)
		if err != nil {
			return err
		}
		p, err := mgr.Start(opts.LogPath)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "started logging process %d -> %s\n", p.ID, p.LogPath)
		return nil
	case "stop":
		opts, err := cli.ParseProc("stop", cmd.Args)
		if err != nil {
			return err
		}
		p, err := mgr.Stop(opts.ProcessID)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "stopped logging process %d -> %s\n", p.ID, p.LogPath)
		return nil
	case "attach":
		opts, err := cli.ParseProcAndTarget("attach", cmd.Args)
		if err != nil {
			return err
		}
		_, sess, err := mgr.Attach(opts.ProcessID, opts.Target)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "attached session %s\n", sess)
		return nil
	case "detach":
		opts, err := cli.ParseProcAndTarget("detach", cmd.Args)
		if err != nil {
			return err
		}
		_, sess, err := mgr.Detach(opts.ProcessID, opts.Target)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "detached session %s\n", sess)
		return nil
	case "_ingest":
		opts, err := cli.ParseIngest(cmd.Args)
		if err != nil {
			return err
		}
		return mgr.Ingest(opts.ProcessID, opts.Session, opts.Pane, stdin)
	case "_attach-session-panes":
		opts, err := cli.ParseAttachSessionPanes(cmd.Args)
		if err != nil {
			return err
		}
		return mgr.AttachSessionPanes(opts.ProcessID, opts.Session)
	default:
		return fmt.Errorf("unknown command %q", cmd.Name)
	}
}
