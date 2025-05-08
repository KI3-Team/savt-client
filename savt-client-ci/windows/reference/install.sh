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

# 以管理员权限运行 PowerShell

# 定义安装目录
$installDir = "C:\Program Files\savt-client"

# 导航到安装目录
Set-Location -Path $installDir

# 1. 创建服务
.\nssm.exe install "savt-client.savt-client-worker" "$installDir\savt-client-worker.exe"

# 2. 设置服务显示名称
.\nssm.exe set "savt-client.savt-client-worker" DisplayName "SAV Client Worker"

# 3. 设置服务描述
.\nssm.exe set "savt-client.savt-client-worker" Description "SAV Client Worker Service"

# 4. 设置服务为开机自动启动
.\nssm.exe set "savt-client.savt-client-worker" Start SERVICE_AUTO_START

# 5. 启动服务
Start-Service -Name "savt-client.savt-client-worker"