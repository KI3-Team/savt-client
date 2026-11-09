#!/bin/bash
set -e

# Stop savt-client-upgrader LaunchDaemon during uninstall
PLIST_PATH="/Library/LaunchDaemons/savt-client.savt-client-upgrader.plist"
SERVICE_LABEL="savt-client.savt-client-upgrader"

if [[ -f "$PLIST_PATH" ]]; then
  launchctl stop "$SERVICE_LABEL" 2>/dev/null || true
  launchctl unload "$PLIST_PATH" 2>/dev/null || true
  rm -f "$PLIST_PATH"
  echo "Removed $SERVICE_LABEL"
fi
