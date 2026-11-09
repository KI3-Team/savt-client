# Copyright 2025 KI3
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/bin/bash

set -e

# ==========================================
# Script Name: install_savt_client_upgrader.sh
# Function: Install savt-client-upgrader as a LaunchDaemon on macOS
# ==========================================

PLIST_PATH="/Library/LaunchDaemons/savt-client.savt-client-upgrader.plist"
SERVICE_LABEL="savt-client.savt-client-upgrader"
EXECUTABLE_PATH="/Applications/savt-client.app/Contents/MacOS/savt-client-upgrader"
LOG_FILE="/tmp/savt-client/install_savt_client_upgrader.log"
exec > >(tee -i "$LOG_FILE") 2>&1

if [[ $EUID -ne 0 ]]; then
   echo "Error: This script must be run with root privileges. Please use sudo."
   exit 1
fi

if [[ ! -f "$EXECUTABLE_PATH" ]]; then
    echo "Error: Executable $EXECUTABLE_PATH does not exist. Please ensure the path is correct."
    exit 1
fi

mkdir -p "/Library/Application Support/savt-client"

cat <<EOF > "$PLIST_PATH"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>$SERVICE_LABEL</string>
    <key>ProgramArguments</key>
    <array>
      <string>$EXECUTABLE_PATH</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Library/Application Support/savt-client/savt-client-upgrader.log</string>
    <key>StandardErrorPath</key>
    <string>/Library/Application Support/savt-client/savt-client-upgrader.err</string>
    <key>UserName</key>
    <string>root</string>
  </dict>
</plist>
EOF

chown root:wheel "$PLIST_PATH"
chmod 644 "$PLIST_PATH"
chmod +x "$EXECUTABLE_PATH"

launchctl unload "$PLIST_PATH" 2>/dev/null || true
launchctl load "$PLIST_PATH"
launchctl stop "$SERVICE_LABEL" 2>/dev/null || true
launchctl start "$SERVICE_LABEL"

echo "Service $SERVICE_LABEL installed and started."
