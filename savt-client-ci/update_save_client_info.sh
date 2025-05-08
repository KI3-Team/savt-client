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

# Set default parameters
OS_LIST=("Windows" "MacOS" "Linux")    # List of supported operating systems
ARCH_LIST=("amd64" "arm64")           # List of supported architectures
VERSION="${1:-2.0.0}"                 # Default version, assumed to be 1.0.0
ENV="${2:test}"
OUTPUT_DIR="${3:-../build/release}"    # Default build output directory
S3_BUCKET="s3://your-bucket-name"     # S3 bucket address, needs to be modified to your actual address
BASE_DIR=$(cd `dirname $0` ; pwd)
mkdir -p ${OUTPUT_DIR}

# Iterate over operating system and architecture combinations
for OS in "${OS_LIST[@]}"; do
    for ARCH in "${ARCH_LIST[@]}"; do
        if [[ "$OS" == "MacOS" ]]; then
            FILE_NAME="savt-client-${VERSION}-mac-${ARCH}.dmg"
            FILE_FORMAT="dmg"
        elif [[ "$OS" == "Linux" ]]; then
            FILE_NAME="savt-client-${VERSION}-linux-${ARCH}.tar.gz"
            FILE_FORMAT="tar.gz"
        elif [[ "$OS" == "Windows" ]]; then
            FILE_NAME="savt-client-${VERSION}-windows-${ARCH}.exe"
            FILE_FORMAT="exe"
        else
            echo "Unsupported OS: $OS"
            continue
        fi
        s3cmd get "s3://ki3-frontend-static/${ENV}/sav/dist/savt-client-${VERSION}/${FILE_NAME}" "$OUTPUT_DIR/" --force
        # Check if the file was downloaded successfully
        FILE_PATH=$OUTPUT_DIR/$FILE_NAME
        if [[ ! -f "$FILE_PATH" ]]; then
            echo "File not found: $FILE_PATH"
            continue
        fi
        # Get file size and SHA256 checksum
        FILE_SIZE_BYTES=$(stat -f %z "$FILE_PATH")
        FILE_SIZE=$(echo "scale=1; $FILE_SIZE_BYTES/1024/1024" | bc)
        FILE_SHA=$(shasum -a 256 "$FILE_PATH" | cut -d ' ' -f 1)
#        # Update the Python script for the document
        python ${BASE_DIR}/update_save_client_info.py \
            --action upsert \
            --version "$VERSION" \
            --arch "$ARCH" \
            --size "${FILE_SIZE} MiB" \
            --sha "$FILE_SHA" \
            --file_format "$FILE_FORMAT" \
            --os "$OS" \
            --output_path "$OUTPUT_DIR"
        echo "Document update completed for $OS $ARCH with version $VERSION."
    done
done

python ${BASE_DIR}/update_save_client_info.py \
--action adjust \
--output_path "$OUTPUT_DIR"