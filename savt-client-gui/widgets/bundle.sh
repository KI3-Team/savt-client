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

# Get the current script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
RESOURCE_DIR="$SCRIPT_DIR/resource"
OUTPUT_DIR="$SCRIPT_DIR"
BUNDLED_FILE="$OUTPUT_DIR/bundled.go"

# Check if fyne tool is installed
if ! command -v fyne &> /dev/null; then
    echo "Error: fyne tool is not installed"
    echo "Please install it using: go install fyne.io/fyne/v2/cmd/fyne@latest"
    exit 1
fi

# Check if resource directory exists
if [ ! -d "$RESOURCE_DIR" ]; then
    echo "Error: Resource directory not found: $RESOURCE_DIR"
    exit 1
fi

# Ensure output directory exists
if [ ! -d "$OUTPUT_DIR" ]; then
    mkdir -p "$OUTPUT_DIR"
fi

# Delete old bundled.go file
if [ -f "$BUNDLED_FILE" ]; then
    rm "$BUNDLED_FILE"
fi

# Create new bundled.go file and write package declaration and import statements
cat > "$BUNDLED_FILE" << EOL
package widgets

import "fyne.io/fyne/v2"
EOL

# Iterate through all .png files in resource directory and process them one by one
find "$RESOURCE_DIR" -name "*.png" | while read -r file; do
    filename=$(basename "$file")
    echo "Processing: $filename"
    
    # Package single file and append to bundled.go
    fyne bundle "$file" >> "$BUNDLED_FILE"
done

# Check if any files were found
if [ ! -s "$BUNDLED_FILE" ]; then
    echo "Error: No files were processed"
    exit 1
fi

# Check if bundled.go was generated successfully
if [ -f "$BUNDLED_FILE" ]; then
    echo "Successfully generated bundled.go"
else
    echo "Error: Failed to generate bundled.go"
    exit 1
fi
