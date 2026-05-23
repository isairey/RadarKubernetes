#!/bin/bash
# Stop a running visual test Radar instance and open the screenshot folder.
# Reads state from .playwright-mcp/visual-test-state.env

set -euo pipefail

STATEFILE=".playwright-mcp/visual-test-state.env"

if [[ ! -f "$STATEFILE" ]]; then
  echo "No visual test running (state file not found: $STATEFILE)" >&2
  exit 1
fi

source "$STATEFILE"

# Kill Radar
if kill "$RADAR_PID" 2>/dev/null; then
  echo "Radar (PID $RADAR_PID) stopped."
else
  echo "Radar (PID $RADAR_PID) was already stopped."
fi

# Open screenshot folder
if [[ -d "$SCREENSHOT_DIR" ]]; then
  echo "Screenshots: $SCREENSHOT_DIR"
  open "$SCREENSHOT_DIR" 2>/dev/null || true
fi

echo "Logs: $RADAR_LOG"

# Clean up state file
rm -f "$STATEFILE"
