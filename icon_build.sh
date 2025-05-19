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

# Define input and output paths
ICON_INPUT="icon.ico" # Input .ico file
ICONSET_DIR="savt-client.iconset" # Intermediate .iconset directory
ICNS_OUTPUT="savt-client.icns" # Final output .icns file

# Check if input file exists
if [ ! -f "$ICON_INPUT" ]; then
    echo "Error: Input file $ICON_INPUT not found"
    exit 1
fi

# Create .iconset directory
mkdir -p "$ICONSET_DIR"

# Convert .ico to multi-resolution .png files (using magick)
magick convert "$ICON_INPUT" -define icon:auto-resize=16,32,64,128,256,512,1024 "$ICONSET_DIR/icon_%d.png"

# Generate .icns file
iconutil -c icns "$ICONSET_DIR" -o "$ICNS_OUTPUT"

# Optional: Remove temporary .iconset directory
rm -rf "$ICONSET_DIR"

echo "All done!"