# tmxlog

`tmxlog` is a tmux-first logging CLI for Ubuntu-style terminal workflows.
It wraps `tmux pipe-pane` and lets you manage logging in a predictable process model:

- **start / stop** = lifecycle of a logging process
- **attach / detach** = include/exclude tmux sessions from a logging process

---

## Why tmxlog

Traditional tools like `script` are great for single-shell capture.
`tmxlog` is built for tmux users who need:

- merged logs from multiple tmux sessions and panes
- explicit separation between process lifecycle and session inclusion
- process IDs (not just log-path-centric behavior)
- automatic handling for newly created panes in attached sessions

---

## Core semantics

1. `start <log_path>` creates a logging process only.
2. `start` does **not** auto-attach any session.
3. `attach` / `detach` support:
   - explicit session targeting with `-t` / `--target`
   - implicit current session targeting when run inside tmux without `-t`
4. Process IDs are auto-indexed (next free numeric ID).
5. If `-p` is omitted on `attach` / `detach` / `stop`, tmxlog falls back to the most recently created active process.
6. `stop` marks the process inactive and detaches attached sessions.

---

## Installation

For full beginner-friendly Ubuntu setup, see:

- [`docs/SETUP_UBUNTU.md`](docs/SETUP_UBUNTU.md)

Quick build:

```bash
go build -o tmxlog ./cmd/tmxlog
```

Optional global install:

```bash
sudo install -m 0755 ./tmxlog /usr/local/bin/tmxlog
```

---

## Complete usage guide

### Command summary

```bash
tmxlog start <log_path>
tmxlog stop [-p <process_id>]
tmxlog attach [-p <process_id>] [-t <tmux_target>]
tmxlog detach [-p <process_id>] [-t <tmux_target>]
```

Internal commands (invoked by tmux integration):

```bash
tmxlog _ingest --process <id> --session <name> --pane <pane_id>
tmxlog _attach-session-panes --process <id> --session <name>
```

### 1) Start a new logging process

```bash
tmxlog start ~/tmx-logs/team.log
```

Example output:

```text
started logging process 0 -> /home/user/tmx-logs/team.log
```

### 2) Attach sessions to the process

Attach by target name/index:

```bash
tmxlog attach -p 0 -t dev
tmxlog attach -p 0 -t 1
```

Attach current session from inside tmux:

```bash
tmxlog attach -p 0
```

### 3) Detach sessions

```bash
tmxlog detach -p 0 -t dev
```

Or, from inside tmux for current session:

```bash
tmxlog detach -p 0
```

### 4) Stop the process

```bash
tmxlog stop -p 0
```

When omitted, `-p` falls back to most recent active process:

```bash
tmxlog stop
```

### 5) Default-process fallback examples

```bash
tmxlog start /tmp/a.log   # process 0
tmxlog start /tmp/b.log   # process 1 (latest)
tmxlog attach -t dev      # attaches dev to process 1
tmxlog detach -t dev      # detaches from process 1
tmxlog stop               # stops process 1
```

---

## Output format and behavior

Each ingested line is written with metadata:

```text
<timestamp> [proc:<id>] [session:<name>] [pane:<pane_id>] <raw pane line>
```

This gives merged, chronological, multi-session logs with source context.

tmxlog applies `pipe-pane -o` to panes in attached sessions and installs tmux hooks so panes created later in those sessions are automatically included.

---

## State management

Default state path:

```text
~/.local/state/tmxlog/state.json
```

Override state file:

```bash
export TMXLOG_STATE_FILE=/custom/path/state.json
```

---

## Architecture overview

- `cmd/tmxlog/main.go`: binary entry point
- `internal/app`: command routing and orchestration
- `internal/cli`: argument parsing/validation
- `internal/process`: lifecycle/session/ingestion core logic
- `internal/state`: persisted process/session metadata store
- `internal/tmux`: tmux command wrapper
- `internal/util`: environment and path helpers

This layered design keeps responsibilities narrow and implementation maintainable.

---

## Developer checks

```bash
gofmt -w ./cmd ./internal
go test ./...
go build ./cmd/tmxlog
```
