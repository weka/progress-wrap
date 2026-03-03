#!/bin/bash
# Simulates a status command that speeds up then slows down.
# Progress follows a raised-cosine (sine-bell velocity) curve over 120 seconds:
#   progress(t) = (1 - cos(π * t / T)) / 2 * 100
# Velocity peaks at t=60s and falls back to zero at t=120s.
#
# Usage: called once per iteration by the demo record loop.

START_FILE="/tmp/pw-sim-start"
T=120

if [ ! -f "$START_FILE" ]; then
    date +%s > "$START_FILE"
fi

start=$(cat "$START_FILE")
now=$(date +%s)
i=$((now - start))
if [ "$i" -ge "$T" ]; then i=$T; fi

pct=$(awk "BEGIN { pi = 3.14159265358979; printf \"%.2f\", (1 - cos(pi * $i / $T)) / 2 * 100 }")
echo "Progress: ${pct}%"
