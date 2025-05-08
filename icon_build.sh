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

# 定义输入和输出路径
ICON_INPUT="icon.ico" # 输入的 .ico 文件
ICONSET_DIR="savt-client.iconset" # 中间生成的 .iconset 文件夹
ICNS_OUTPUT="savt-client.icns" # 最终输出的 .icns 文件

# 检查输入文件是否存在
if [[ ! -f "$ICON_INPUT" ]]; then
  echo "Error: Input file '$ICON_INPUT' not found."
  exit 1
fi

# 创建 .iconset 文件夹
echo "Creating .iconset directory..."
mkdir -p "$ICONSET_DIR"

# 将 .ico 转换为多分辨率 .png 文件（使用 magick）
echo "Converting .ico to .png at different resolutions..."
magick "$ICON_INPUT" -resize 16x16 "$ICONSET_DIR/icon_16x16.png"
magick "$ICON_INPUT" -resize 32x32 "$ICONSET_DIR/icon_32x32.png"
magick "$ICON_INPUT" -resize 64x64 "$ICONSET_DIR/icon_64x64.png"
magick "$ICON_INPUT" -resize 128x128 "$ICONSET_DIR/icon_128x128.png"
magick "$ICON_INPUT" -resize 256x256 "$ICONSET_DIR/icon_256x256.png"
magick "$ICON_INPUT" -resize 512x512 "$ICONSET_DIR/icon_512x512.png"

# 生成 .icns 文件
echo "Generating .icns file from .iconset..."
if iconutil -c icns -o "$ICNS_OUTPUT" "$ICONSET_DIR"; then
  echo "Icon successfully built: $ICNS_OUTPUT"
else
  echo "Error: Failed to generate .icns file."
  exit 1
fi

# 可选：删除临时的 .iconset 文件夹
# echo "Cleaning up temporary .iconset directory..."
# rm -rf "$ICONSET_DIR"

echo "All done!"