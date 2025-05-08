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
# Script Name: stop_savt_client_worker.sh
# Function: Stop savt-client-worker as a system-level service on macOS
# ==========================================

# Configuration Variables
PLIST_PATH="/Library/LaunchDaemons/savt-client.savt-client-worker.plist"
SERVICE_LABEL="savt-client.savt-client-worker"
EXECUTABLE_PATH="/Applications/savt-client.app/Contents/MacOS/savt-client-worker"
LOG_FILE="/tmp/savt-client/stop_savt_client_worker.log"
exec > >(tee -i "$LOG_FILE") 2>&1

# Get the currently logged-in user's username
CURRENT_USER=$(stat -f "%Su" /dev/console)
USER_HOME=$(eval echo ~$CURRENT_USER)

# Check if the script is run with root privileges
if [[ $EUID -ne 0 ]]; then
   echo "Error: This script must be run with root privileges. Please use sudo."
   exit 1
fi

# Check if the service is loaded
if launchctl list | grep -q "$SERVICE_LABEL"; then
    echo "Stopping service $SERVICE_LABEL..."
    if ! launchctl stop "$SERVICE_LABEL"; then
        echo "Warning: Failed to stop service $SERVICE_LABEL. It might not be running."
    else
        echo "Service $SERVICE_LABEL stopped successfully."
    fi
else
    echo "Info: Service $SERVICE_LABEL is not currently loaded."
fi

# Unload the service
echo "Unloading service $SERVICE_LABEL..."
if ! launchctl unload "$PLIST_PATH" 2>/dev/null; then
    echo "Warning: Failed to unload service $SERVICE_LABEL. It might not be loaded."
else
    echo "Service $SERVICE_LABEL unloaded successfully."
fi

# Remove the plist file
if [ -f "$PLIST_PATH" ]; then
    echo "Removing plist file $PLIST_PATH..."
    rm -f "$PLIST_PATH"
    echo "Plist file removed."
else
    echo "Info: Plist file $PLIST_PATH does not exist."
fi

# Verify the service status
if launchctl list | grep -q "$SERVICE_LABEL"; then
    echo "Error: Service $SERVICE_LABEL is still running or in an abnormal state."
else
    echo "Service $SERVICE_LABEL has been successfully stopped and unloaded."
fi

