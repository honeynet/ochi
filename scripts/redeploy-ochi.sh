#!/bin/bash
# Server-side helper: replace the Ochi binary and restart the "ochi" screen session.
# Expects a new binary at /tmp/ochi.new (build with `make build`, then scp the
# `server` binary there from the workstation).
# Fails loudly if the old process cannot be stopped or the new one does not stay up.
# Paths and timings can be overridden through the environment.
set -euo pipefail

NAME=${OCHI_NAME:-ochi}
OPT_DIR=${OCHI_DIR:-/opt/$NAME}
BIN="$OPT_DIR/$NAME"
NEW=${OCHI_NEW:-/tmp/$NAME.new}
CONFIG=${OCHI_CONFIG:-$OPT_DIR/config.yaml}
STOP_TIMEOUT=${OCHI_STOP_TIMEOUT:-20} # seconds to wait for a graceful exit before SIGKILL
START_WAIT=${OCHI_START_WAIT:-5}      # seconds the new process must survive

die() {
	echo "$*" >&2
	exit 1
}

running() {
	pgrep -x "$NAME" >/dev/null
}

stop_old() {
	# Quit every matching screen session, not just one.
	screen -ls 2>/dev/null | awk -v n="\\\\.$NAME[[:space:]]" '$1 ~ n {print $1}' | while read -r s; do
		screen -S "$s" -X quit 2>/dev/null || true
	done
	pkill -TERM -x "$NAME" 2>/dev/null || true

	for ((i = 0; i < STOP_TIMEOUT; i++)); do
		running || break
		sleep 1
	done
	if running; then
		echo "$NAME still running after ${STOP_TIMEOUT}s; sending SIGKILL" >&2
		pkill -KILL -x "$NAME" || true
		sleep 1
	fi
	running && die "failed to stop the old $NAME process"
	screen -wipe >/dev/null 2>&1 || true
}

start_new() {
	cd "$OPT_DIR"
	screen -dmS "$NAME" "./$NAME" -config "$CONFIG"

	sleep "$START_WAIT"
	PID=$(pgrep -x "$NAME" | head -n 1 || true)
	[[ -n "$PID" ]] || die "$NAME exited within ${START_WAIT}s of starting; check the config and port (run ./$NAME -config $CONFIG by hand)"
}

[[ -f "$NEW" ]] || die "missing $NEW; scp the binary there first"

chmod 755 "$NEW"
stop_old
install -m 755 "$NEW" "$BIN"
rm -f "$NEW"
start_new

echo "redeployed: pid $PID, started $(ps -o lstart= -p "$PID")"
screen -ls || true
