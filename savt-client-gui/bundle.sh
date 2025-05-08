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

# Get the directory where the current script is located
script_dir="$(cd "$(dirname "$0")" && pwd)"
resource_dir="${script_dir}/resources"
output_dir="${script_dir}"
output_file="${output_dir}/bundled.go"

# Check if fyne tool is installed
if ! command -v fyne &> /dev/null; then
  echo "Fyne CLI tool is not installed. Installing now..."
  go install fyne.io/fyne/v2/cmd/fyne@latest
  if ! command -v fyne &> /dev/null; then
    echo "Error: Failed to install Fyne CLI tool. Please check your Go environment."
    exit 1
  fi
fi

# Check if resource directory exists
if [ ! -d "$resource_dir" ]; then
  echo "Error: Resource directory not found: $resource_dir"
  exit 1
fi

# Ensure output directory exists
if [ ! -d "$output_dir" ]; then
  mkdir -p "$output_dir"
  echo "Created output directory: $output_dir"
fi

# Delete old bundled.go file
if [ -f "$output_file" ]; then
  rm -f "$output_file"
  echo "Removed old $output_file"
fi

# Create new bundled.go file and write package declaration and import statements
echo "package main" > "$output_file"
echo >> "$output_file"
echo "import (" >> "$output_file"
echo "	\"fyne.io/fyne/v2\"" >> "$output_file"
echo ")" >> "$output_file"
echo >> "$output_file"

# Process all .png files in the resource directory one by one
cd "$resource_dir" || exit
found_files=0
for file in *.png; do
  if [ -f "$file" ]; then
    # Bundle single file and append to bundled.go
    fyne bundle --append -o "$output_file" "$file"
    echo "Processed: $file"
    found_files=1
  fi
done

# Check if any files were found
if [ $found_files -eq 0 ]; then
  echo "Error: No .png files found in $resource_dir"
  exit 1
fi

# Check if bundled.go was successfully generated
if [ -f "$output_file" ]; then
  echo "Resources bundled into $output_file"
else
  echo "Error: Failed to create $output_file"
  exit 1
fi