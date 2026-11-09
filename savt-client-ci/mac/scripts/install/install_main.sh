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
# Script name: install_main.sh
# Function: Handle the main installation logic for savt-client
# ==========================================

# Get current logged-in user's username and home directory
CURRENT_USER=$(stat -f '%Su' /dev/console)
USER_HOME=$(eval echo ~"$CURRENT_USER")
# Define source file path and target path
SOURCE_FILE="/Applications/savt-client.app/Contents/MacOS/service.json"
TARGET_DIR="/Library/Application Support/savt-client"
# Log file
mkdir -p /tmp/savt-client
LOG_FILE="/tmp/savt-client/install_main.log"
exec > >(tee -i "$LOG_FILE") 2>&1

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
    echo "ERROR: Source file not found at $SOURCE_FILE"
    exit 1
fi
# Create target directory
mkdir -p "$TARGET_DIR"
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to create directory $TARGET_DIR"
    exit 1
fi

# Copy file to target directory
cp "$SOURCE_FILE" "$TARGET_DIR/"
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to copy $SOURCE_FILE to $TARGET_DIR"
    exit 1
fi

# End log
echo "install_main.sh script completed successfully!" >> "$LOG_FILE"
exit 0
