#!/bin/bash
# Records a 2-minute demo of progress-wrap wrapping the sine-curve simulation.
# Mimics the idiomatic usage loop:
#   while true; do ./progress-wrap cmd | tee -a log > display; clear; cat display; sleep 1; done
#
# Run from the project root: bash demo/record.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PW="$PROJECT_ROOT/progress-wrap"
SIM="$SCRIPT_DIR/sim.sh"
STATE_FILE="/tmp/pw-demo.state"
LOG_FILE="/tmp/pw-demo.log"
DISPLAY_FILE="/tmp/pw-demo-display"
STEPS=120

# Clean up leftover state from any previous run
rm -f "$STATE_FILE" "$LOG_FILE" "$DISPLAY_FILE" /tmp/pw-sim-start

chmod +x "$SIM"

for i in $(seq 1 "$STEPS"); do
    "$PW" --state "$STATE_FILE" "$SIM" | tee -a "$LOG_FILE" > "$DISPLAY_FILE"
    clear
    cat "$DISPLAY_FILE"
    sleep 1
done
