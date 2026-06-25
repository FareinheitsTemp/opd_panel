#!/usr/bin/env bash
set -euo pipefail

echo "==> OPD dev mode"

export $(grep -v '^#' .env | xargs)

(cd agent && cargo run) &
AGENT_PID=$!
echo "  Agent PID: $AGENT_PID"

(cd panel && go run ./cmd/opdd) &
DAEMON_PID=$!
echo "  Daemon PID: $DAEMON_PID"

trap "kill $AGENT_PID $DAEMON_PID 2>/dev/null; echo 'Stopped.'" EXIT INT TERM
wait
