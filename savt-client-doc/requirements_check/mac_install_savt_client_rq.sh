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

if [ "$(uname)" != "Darwin" ]; then
    echo "This script is intended for macOS."
    exit 1
fi

if ! command -v brew >/dev/null 2>&1; then
    echo "Homebrew not found. Please install Homebrew."
    exit 1
fi

if brew ls --versions libpcap >/dev/null 2>&1; then
    echo "libpcap is installed via Homebrew."
    exit 0
fi

echo "libpcap not found, installing via Homebrew..."
brew update
brew install libpcap
echo "Installation complete!"
