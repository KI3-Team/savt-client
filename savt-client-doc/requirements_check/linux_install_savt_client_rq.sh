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
ldconfig -p | grep -q libpcap && { echo "libpcap is installed."; exit 0; }
echo "libpcap not found, installing..."
if command -v apt-get >/dev/null; then
    sudo apt-get update && sudo apt-get install -y libpcap-dev;
elif command -v yum >/dev/null; then
    sudo yum install -y libpcap-devel;
elif command -v dnf >/dev/null; then
    sudo dnf install -y libpcap-devel;
elif command -v pacman >/dev/null; then
    sudo pacman -Sy --noconfirm libpcap;
elif command -v zypper >/dev/null; then
    sudo zypper install -y libpcap-devel;
else
    echo "Unrecognized package manager. Please install libpcap manually." && exit 1;
fi
echo "Installation complete!"
