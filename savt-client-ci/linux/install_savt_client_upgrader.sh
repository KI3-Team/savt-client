#!/bin/bash
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

set -e

# ==========================================
# Install or uninstall savt-client-upgrader as a systemd service on Linux
# ==========================================

SERVICE_NAME="savt-client-upgrader"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
EXECUTABLE_NAME_PATTERN="savt-client-upgrader*"
INSTALL_DIR="/usr/local/bin"
LOG_DIR="/var/log/savt-client"
LOG_FILE="${LOG_DIR}/install_savt_client_upgrader.log"
WORKING_DIR="${INSTALL_DIR}"

mkdir -p "${LOG_DIR}"
exec > >(tee -i "${LOG_FILE}") 2>&1
shopt -s nullglob

check_root() {
    if [[ "$EUID" -ne 0 ]]; then
        echo "Error: This script must be run as root or with sudo." >&2
        exit 1
    fi
}

install_executable() {
    matched_files=(./${EXECUTABLE_NAME_PATTERN})
    if [[ ${#matched_files[@]} -eq 0 ]]; then
        # Also accept already-renamed binary without version suffix in package root.
        if [[ -f ./savt-client-upgrader ]]; then
            matched_files=(./savt-client-upgrader)
        else
            echo "Error: No savt-client-upgrader executable found." >&2
            exit 1
        fi
    fi
    if [[ ${#matched_files[@]} -gt 1 ]]; then
        echo "Error: Multiple savt-client-upgrader executables found." >&2
        exit 1
    fi
    exe="${matched_files[0]}"
    cp "${exe}" "${INSTALL_DIR}/savt-client-upgrader"
    chmod +x "${INSTALL_DIR}/savt-client-upgrader"
    EXECUTABLE_PATH="${INSTALL_DIR}/savt-client-upgrader"
    echo "Installed ${EXECUTABLE_PATH}"
}

create_service_file() {
    cat <<EOF > "${SERVICE_FILE}"
[Unit]
Description=SAV Client Upgrader Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${EXECUTABLE_PATH}
WorkingDirectory=${WORKING_DIR}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
}

uninstall() {
    if systemctl is-active --quiet "${SERVICE_NAME}.service" 2>/dev/null; then
        systemctl stop "${SERVICE_NAME}.service" || true
    fi
    if systemctl is-enabled --quiet "${SERVICE_NAME}.service" 2>/dev/null; then
        systemctl disable "${SERVICE_NAME}.service" || true
    fi
    rm -f "${SERVICE_FILE}"
    systemctl daemon-reload || true
    rm -f "${INSTALL_DIR}/savt-client-upgrader"
    # Keep /var/lib/savt-client/device.id across uninstall/reinstall.
    rm -f "${INSTALL_DIR}/savt-client.device.id" || true
    echo "savt-client-upgrader uninstalled."
}

main() {
    check_root
    case "$1" in
        install)
            uninstall || true
            install_executable
            create_service_file
            systemctl daemon-reload
            systemctl enable "${SERVICE_NAME}.service"
            systemctl start "${SERVICE_NAME}.service"
            systemctl status "${SERVICE_NAME}.service" --no-pager || true
            echo "savt-client-upgrader service installed and started."
            ;;
        uninstall)
            uninstall
            ;;
        *)
            echo "Usage: $0 {install|uninstall}"
            exit 1
            ;;
    esac
}

main "$@"
