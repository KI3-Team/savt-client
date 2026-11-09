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

# Run PowerShell with administrator privileges

# Define installation directory
$installDir = "C:\Program Files\savt-client"

# Navigate to installation directory
Set-Location -Path $installDir

# 1. Create service
.\nssm.exe install "savt-client.savt-client-worker" "$installDir\savt-client-worker.exe"

# 2. Set service display name
.\nssm.exe set "savt-client.savt-client-worker" DisplayName "SAV Client Worker"

# 3. Set service description
.\nssm.exe set "savt-client.savt-client-worker" Description "SAV Client Worker Service"

# 4. Set service to start automatically at boot
.\nssm.exe set "savt-client.savt-client-worker" Start SERVICE_AUTO_START

# 5. Start service
Start-Service -Name "savt-client.savt-client-worker"