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
# Script Name: start_sav_client_worker.sh
# Function: Start the already loaded savt-client-worker service
# ==========================================

# Configuration Variables
SERVICE_LABEL="savt-client.savt-client-worker"
PLIST_PATH="/Library/LaunchDaemons/savt-client.savt-client-worker.plist"

# Check if the script is run with root privileges
if [[ $EUID -ne 0 ]]; then
   echo "Error: This script must be run with root privileges. Please use sudo."
   exit 1
fi

# Check if the Plist file exists
if [[ ! -f "$PLIST_PATH" ]]; then
    echo "Error: Plist file $PLIST_PATH does not exist. Please run the installation script first."
    exit 1
fi

# Load the service
echo "Loading service..."
launchctl unload "$PLIST_PATH" 2>/dev/null || true
launchctl load "$PLIST_PATH"
echo "Service loaded."

# Start the service
echo "Starting service..."
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