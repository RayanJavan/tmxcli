#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
BIN_DIR="$TMP_DIR/bin"
mkdir -p "$BIN_DIR"

cleanup() {
  tmux kill-session -t tmxlog-e2e >/dev/null 2>&1 || true
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

pushd "$ROOT_DIR" >/dev/null
GOFLAGS="" go build -o "$BIN_DIR/tmxlog" ./cmd/tmxlog
popd >/dev/null

export TMXLOG_STATE_FILE="$TMP_DIR/state.json"
TMXLOG_BIN="$BIN_DIR/tmxlog"

LOG_FILE="$TMP_DIR/test.log"

# Start should immediately create log file.
"$TMXLOG_BIN" start "$LOG_FILE"
[[ -f "$LOG_FILE" ]]
grep -q "started logging process" "$LOG_FILE"

# Create a detached tmux session and attach logging.
tmux new-session -d -s tmxlog-e2e
sleep 0.2
"$TMXLOG_BIN" attach -t tmxlog-e2e

# Emit output from the pane.
tmux send-keys -t tmxlog-e2e:0.0 'echo TMXLOG_E2E_MESSAGE' C-m
sleep 0.8

"$TMXLOG_BIN" stop
sleep 0.2

grep -q "TMXLOG_E2E_MESSAGE" "$LOG_FILE"
grep -q "stopped logging process" "$LOG_FILE"

echo "E2E PASS"
