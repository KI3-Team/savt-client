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
# Script Name: install_savt_client_worker.sh
# Function: Install savt-client-worker as a system-level service on macOS and load the service
# ==========================================

# Configuration Variables
PLIST_PATH="/Library/LaunchDaemons/savt-client.savt-client-worker.plist"
SERVICE_LABEL="savt-client.savt-client-worker"
EXECUTABLE_PATH="/Applications/savt-client.app/Contents/MacOS/savt-client-worker"
LOG_FILE="/tmp/savt-client/install_savt_client_worker.log"
exec > >(tee -i "$LOG_FILE") 2>&1

# Get the currently logged-in user's username
CURRENT_USER=$(stat -f "%Su" /dev/console)
USER_HOME=$(eval echo ~$CURRENT_USER)


# Check if the script is run with root privileges
if [[ $EUID -ne 0 ]]; then
   echo "Error: This script must be run with root privileges. Please use sudo."
   exit 1
fi

# Check if the executable exists
if [[ ! -f "$EXECUTABLE_PATH" ]]; then
    echo "Error: Executable $EXECUTABLE_PATH does not exist. Please ensure the path is correct."
    exit 1
fi

# Check if the service already exists
if launchctl list | grep -q "$SERVICE_LABEL"; then
    echo "Warning: Service $SERVICE_LABEL already exists and may be installed."
fi


# Create the Plist file content
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
    <key>StandardOutPath</key>
    <string>/Library/Application Support/savt-client/savt-client-worker.log</string>
    <key>StandardErrorPath</key>
    <string>/Library/Application Support/savt-client/savt-client-worker.err</string>
    <key>UserName</key>
    <string>root</string>
  </dict>
</plist>
EOF

echo "Plist file created: $PLIST_PATH"
chown root:wheel "$PLIST_PATH"
chmod 644 "$PLIST_PATH"
echo "Set plist file ownership to root:wheel and permissions to 644."

# Ensure the executable has execute permissions
chmod +x "$EXECUTABLE_PATH"
echo "Set execute permissions on the executable."

# Load the service
echo "Loading service..."
launchctl unload "$PLIST_PATH" 2>/dev/null || true
launchctl load "$PLIST_PATH"
echo "Service loaded."

# Start the service
echo "ReStarting service..."
launchctl stop "$SERVICE_LABEL"
launchctl start "$SERVICE_LABEL"
echo "Service started."

# Verify the service status
SERVICE_STATUS=$(launchctl list | grep "$SERVICE_LABEL" | awk '{print $2}')
if [[ "$SERVICE_STATUS" == "0" ]]; then
    echo "Service $SERVICE_LABEL is running."
else
    echo "Warning: Service $SERVICE_LABEL is in an abnormal state (Status code: $SERVICE_STATUS). Please check the configuration."
fi

echo "Startup script execution completed."
echo "Installation script execution completed. The service will automatically start at boot."
echo "If you need to uninstall the service, please run the following command:"
echo "sudo launchctl unload $PLIST_PATH && sudo rm $PLIST_PATH"