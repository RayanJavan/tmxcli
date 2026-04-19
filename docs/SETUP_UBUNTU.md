# tmxlog Ubuntu Setup Guide (Beginner Friendly)

This guide walks you through installing, building, and using **tmxlog** on an Ubuntu VM from scratch.

---

## 1) What you are installing

`tmxlog` is a command-line tool that integrates with `tmux` using `pipe-pane`.

It gives you:

- `start` / `stop` to manage **logging processes**
- `attach` / `detach` to include/exclude **tmux sessions** in those processes
- merged chronological logs across panes and sessions

---

## 2) Prerequisites

Open a terminal and install required packages:

```bash
sudo apt update
sudo apt install -y tmux golang-go git
```

Verify installation:

```bash
tmux -V
go version
git --version
```

---

## 3) Get the source code

If you already have the repo, skip to step 4.

```bash
git clone <YOUR_REPO_URL>
cd tmxcli
```

If your repo already exists in a folder, just `cd` into it.

---

## 4) Build the tmxlog binary

From repo root:

```bash
go build -o tmxlog ./cmd/tmxlog
```

You now have a local binary at:

```text
./tmxlog
```

Test it quickly:

```bash
./tmxlog
```

You should see a usage error/help-style message (expected if no command is provided).

---

## 5) Install it globally (recommended)

Move binary into your PATH:

```bash
sudo install -m 0755 ./tmxlog /usr/local/bin/tmxlog
```

Verify:

```bash
which tmxlog
tmxlog
```

---

## 6) Start tmux and create test sessions

Start tmux:

```bash
tmux
```

Inside tmux, create or rename sessions as desired.

You can also create sessions from outside tmux:

```bash
tmux new-session -d -s dev
tmux new-session -d -s ops
```

List sessions:

```bash
tmux ls
```

---

## 7) First end-to-end logging workflow

### Step A: Start a logging process

```bash
tmxlog start ~/tmx-logs/demo.log
```

Expected output example:

```text
started logging process 0 -> /home/ubuntu/tmx-logs/demo.log
```

> Important: `start` does **not** auto-attach any sessions.

### Step B: Attach sessions to logging process

Attach by explicit session target:

```bash
tmxlog attach -p 0 -t dev
tmxlog attach -p 0 -t ops
```

Or, from inside a tmux session, attach current session implicitly:

```bash
tmxlog attach -p 0
```

### Step C: Generate activity

In attached tmux panes, run commands like:

```bash
echo "hello from dev"
date
uname -a
```

### Step D: Inspect merged log

```bash
tail -n 50 ~/tmx-logs/demo.log
```

Each line contains timestamp + process/session/pane metadata.

### Step E: Detach one session (optional)

```bash
tmxlog detach -p 0 -t ops
```

### Step F: Stop logging process

```bash
tmxlog stop -p 0
```

This detaches remaining sessions and finalizes that process lifecycle.

---

## 8) Default process fallback behavior

If you do not pass `-p`, `tmxlog` uses the **most recently created active process**.

Example:

```bash
tmxlog start ~/tmx-logs/a.log   # process 0
tmxlog start ~/tmx-logs/b.log   # process 1 (latest)
tmxlog attach -t dev            # applies to process 1
tmxlog stop                     # stops process 1
```

---

## 9) State and files location

By default, process state is stored at:

```text
~/.local/state/tmxlog/state.json
```

Override via environment variable:

```bash
export TMXLOG_STATE_FILE=/custom/path/state.json
```

Log output location is whatever path you pass to `tmxlog start <log_path>`.

---

## 10) Updating to a new version

From repo root:

```bash
git pull
go build -o tmxlog ./cmd/tmxlog
sudo install -m 0755 ./tmxlog /usr/local/bin/tmxlog
```

---

## 11) Troubleshooting

### `tmxlog attach` says no target specified and not running inside tmux

You ran `attach` without `-t` outside tmux.

Fix:

- either run from within tmux, or
- pass target explicitly: `tmxlog attach -t <session>`

### `process not found` or `process is not active`

Use a valid active process ID. If unsure, start a new process and retry.

### No output appears in log

- Confirm session was attached
- Confirm pane activity occurred after attach
- Confirm log path permissions

---

## 12) Quick command reference

```bash
tmxlog start <log_path>
tmxlog stop [-p <process_id>]
tmxlog attach [-p <process_id>] [-t <tmux_target>]
tmxlog detach [-p <process_id>] [-t <tmux_target>]
```

Internal commands (used by hooks/pipe integration):

```bash
tmxlog _ingest --process <id> --session <name> --pane <pane_id>
tmxlog _attach-session-panes --process <id> --session <name>
```
