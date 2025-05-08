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


GO_LICENSE_HEADER="// Copyright __YEAR__ __COPYRIGHT__
//
// Licensed under the Apache License, Version 2.0 (the \"License\");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an \"AS IS\" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License."

SHELL_LICENSE_HEADER="# Copyright __YEAR__ __COPYRIGHT__
#
# Licensed under the Apache License, Version 2.0 (the \"License\");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an \"AS IS\" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License."

YEAR="2025"
COPYRIGHT="KI3"

GO_LICENSE_HEADER="${GO_LICENSE_HEADER/__YEAR__/$YEAR}"
GO_LICENSE_HEADER="${GO_LICENSE_HEADER/__COPYRIGHT__/$COPYRIGHT}"
SHELL_LICENSE_HEADER="${SHELL_LICENSE_HEADER/__YEAR__/$YEAR}"
SHELL_LICENSE_HEADER="${SHELL_LICENSE_HEADER/__COPYRIGHT__/$COPYRIGHT}"

process_file() {
    local file="$1"
    local license_header="$2"
    local comment_pattern="$3"

    if grep -q "Copyright $YEAR $COPYRIGHT" "$file" && grep -q "Apache License" "$file"; then
        return
    fi

    TMP_FILE=$(mktemp)
    
    if grep -q "Copyright" "$file"; then
        START_LINE=$(grep -n -v "$comment_pattern" "$file" | head -n 1 | cut -d: -f1)
        if [ -n "$START_LINE" ]; then
            tail -n +$START_LINE "$file" > "$TMP_FILE"
        else
            touch "$TMP_FILE"
        fi
    else
        cat "$file" > "$TMP_FILE"
    fi
    
    FINAL_TMP=$(mktemp)
    echo "$license_header" > "$FINAL_TMP"
    echo "" >> "$FINAL_TMP"
    cat "$TMP_FILE" >> "$FINAL_TMP"
    rm "$TMP_FILE"
    mv "$FINAL_TMP" "$file"
}

# Process Go files
find . -type f -name "*.go" | while read -r file; do
    process_file "$file" "$GO_LICENSE_HEADER" "//"
done

# Process Shell files
find . -type f -name "*.sh" | while read -r file; do
    process_file "$file" "$SHELL_LICENSE_HEADER" "#"
done

echo "License headers processed successfully."
