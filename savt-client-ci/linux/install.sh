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

# Exit immediately if a command exits with a non-zero status
set -e

# ==========================================
# Script Name: install_sav_client_worker.sh
# Purpose: Install or uninstall savt-client-worker as a systemd service on Linux
# ==========================================

# -------------------------------
# 1. Variables
# -------------------------------
SERVICE_NAME="savt-client-worker"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
EXECUTABLE_NAME_PATTERN="savt-client-worker*"
INSTALL_DIR="/usr/local/bin"
CONFIG_FILE="service.json"
INSTALL_CONFIG_DIR="/var/lib/savt-client"

# Log directory and file
LOG_DIR="/var/log/savt-client"
LOG_FILE="${LOG_DIR}/install_sav_client_worker.log"

# Working directory for the systemd service (modify if needed)
WORKING_DIR="${INSTALL_DIR}"

# -------------------------------
# 2. Create LOG_DIR if needed and set up logging
# -------------------------------
if [[ ! -d "${LOG_DIR}" ]]; then
    mkdir -p "${LOG_DIR}" || {
        echo "Error: Failed to create log directory '${LOG_DIR}'." >&2
        exit 1
    }
fi

# Redirect all output (stdout and stderr) to both the console and the log file
exec > >(tee -i "${LOG_FILE}") 2>&1

# Ensure unmatched globs expand to an empty list instead of the literal pattern
shopt -s nullglob

# -------------------------------
# 3. Function Definitions
# -------------------------------

# Checks if the script is running as root
check_root() {
    if [[ "$EUID" -ne 0 ]]; then
        echo "Error: This script must be run as root or with sudo." >&2
        exit 1
    fi
}

# Checks if systemd is available
check_systemd() {
    if ! command -v systemctl &> /dev/null; then
        echo "Error: systemd is not installed or not available." >&2
        exit 1
    fi
}

# Installs the savt-client-worker executable
install_executable() {
    echo "Installing executable to ${INSTALL_DIR} ..."
    echo "Current directory: $(pwd)"
    ls -l ./${EXECUTABLE_NAME_PATTERN} 2>/dev/null || true

    # Gather all matching files
    matched_files=(./${EXECUTABLE_NAME_PATTERN})

    # Check if any files matched
    if [[ ${#matched_files[@]} -eq 0 ]]; then
        echo "Error: No executables matching pattern './${EXECUTABLE_NAME_PATTERN}' found." >&2
        exit 1
    fi

    # Only one executable should be found
    if [[ ${#matched_files[@]} -gt 1 ]]; then
        echo "Error: Multiple executables found matching '${EXECUTABLE_NAME_PATTERN}'. Please ensure only one executable is present." >&2
        exit 1
    fi

    exe="${matched_files[0]}"
    if [[ -f "${exe}" ]]; then
        echo "Found executable: ${exe}"
        cp "${exe}" "${INSTALL_DIR}/"
        chmod +x "${INSTALL_DIR}/$(basename "${exe}")"
        echo "Executable installed to ${INSTALL_DIR}/$(basename "${exe}")"

        # Set EXECUTABLE_PATH to the newly installed file
        EXECUTABLE_PATH="${INSTALL_DIR}/$(basename "${exe}")"
    else
        echo "Warning: '${exe}' is not a regular file, skipping." >&2
    fi
}

# Installs the configuration file
install_config() {
    echo "Installing configuration file to ${INSTALL_CONFIG_DIR} ..."

    # Check if the config file exists in the current directory
    if [[ ! -f "${CONFIG_FILE}" ]]; then
        echo "Error: Configuration file '${CONFIG_FILE}' not found in the current directory." >&2
        exit 1
    fi

    # Create the config directory if needed
    if [[ ! -d "${INSTALL_CONFIG_DIR}" ]]; then
        echo "Creating configuration directory ${INSTALL_CONFIG_DIR} ..."
        mkdir -p "${INSTALL_CONFIG_DIR}" || {
            echo "Error: Failed to create configuration directory '${INSTALL_CONFIG_DIR}'." >&2
            exit 1
        }
        echo "Configuration directory created."
    fi

    # Copy the config file
    cp "${CONFIG_FILE}" "${INSTALL_CONFIG_DIR}/" || {
        echo "Error: Failed to copy ${CONFIG_FILE} to ${INSTALL_CONFIG_DIR}." >&2
        exit 1
    }
    chmod 644 "${INSTALL_CONFIG_DIR}/${CONFIG_FILE}" || {
        echo "Error: Failed to set permissions on ${INSTALL_CONFIG_DIR}/${CONFIG_FILE}." >&2
        exit 1
    }
    echo "Configuration file installed to ${INSTALL_CONFIG_DIR}/${CONFIG_FILE}"
}

# Creates the systemd service file
create_service_file() {
    echo "Creating systemd service file ${SERVICE_FILE} ..."
    cat <<EOF > "${SERVICE_FILE}"
[Unit]
Description=SAV Client Worker Service
After=network.target

[Service]
Type=simple
ExecStart=${EXECUTABLE_PATH}
WorkingDirectory=${WORKING_DIR}
Restart=on-failure
RestartSec=5
# You can uncomment and customize the following line to set environment variables if needed
# Environment="ENV_VAR1=value1" "ENV_VAR2=value2"

[Install]
WantedBy=multi-user.target
EOF

    echo "Service file created."
}

# Reloads the systemd daemon
reload_systemd() {
    echo "Reloading systemd daemon ..."
    systemctl daemon-reload || {
        echo "Error: Failed to reload systemd daemon." >&2
        exit 1
    }
    echo "Systemd daemon reloaded."
}

# Enables and starts the savt-client-worker service
enable_and_start_service() {
    echo "Enabling ${SERVICE_NAME} to start on boot ..."
    systemctl enable "${SERVICE_NAME}.service" || {
        echo "Error: Failed to enable ${SERVICE_NAME} service." >&2
        exit 1
    }

    echo "Starting ${SERVICE_NAME} service ..."
    systemctl start "${SERVICE_NAME}.service" || {
        echo "Error: Failed to start ${SERVICE_NAME} service." >&2
        exit 1
    }

    echo "Service enabled and started."
}

# Shows the service status
show_service_status() {
    echo "Displaying service status:"
    systemctl status "${SERVICE_NAME}.service" --no-pager || {
        echo "Warning: Failed to retrieve service status." >&2
    }
}

# Uninstalls the savt-client-worker service
uninstall() {
    echo "Uninstalling ${SERVICE_NAME} service ..."

    # Stop the service if it is running
    if systemctl is-active --quiet "${SERVICE_NAME}.service"; then
        echo "Stopping ${SERVICE_NAME} service ..."
        systemctl stop "${SERVICE_NAME}.service" || {
            echo "Error: Failed to stop ${SERVICE_NAME} service." >&2
            exit 1
        }
    fi

    # Disable the service if it is enabled
    if systemctl is-enabled --quiet "${SERVICE_NAME}.service"; then
        echo "Disabling ${SERVICE_NAME} from starting on boot ..."
        systemctl disable "${SERVICE_NAME}.service" || {
            echo "Error: Failed to disable ${SERVICE_NAME} service." >&2
            exit 1
        }
    fi

    # Remove the systemd service file
    if [[ -f "${SERVICE_FILE}" ]]; then
        echo "Removing service file ${SERVICE_FILE} ..."
        rm -f "${SERVICE_FILE}" || {
            echo "Error: Failed to remove service file ${SERVICE_FILE}." >&2
            exit 1
        }
    else
        echo "Service file ${SERVICE_FILE} does not exist, skipping."
    fi

    # Reload systemd to apply changes
    reload_systemd

    # Remove the executable
    if [[ -n "${EXECUTABLE_PATH}" && -f "${EXECUTABLE_PATH}" ]]; then
        echo "Removing executable ${EXECUTABLE_PATH} ..."
        rm -f "${EXECUTABLE_PATH}" || {
            echo "Error: Failed to remove executable ${EXECUTABLE_PATH}." >&2
            exit 1
        }
    else
        echo "Executable ${EXECUTABLE_PATH} does not exist, skipping."
    fi

    # Remove the configuration directory
    if [[ -d "${INSTALL_CONFIG_DIR}" ]]; then
        echo "Removing configuration directory ${INSTALL_CONFIG_DIR} ..."
        rm -rf "${INSTALL_CONFIG_DIR}" || {
            echo "Error: Failed to remove configuration directory ${INSTALL_CONFIG_DIR}." >&2
            exit 1
        }
    else
        echo "Configuration directory ${INSTALL_CONFIG_DIR} does not exist, skipping."
    fi

    # Remove the log directory
    if [[ -d "${LOG_DIR}" ]]; then
        echo "Removing log directory '${LOG_DIR}' ..."
        rm -rf "${LOG_DIR}" || {
            echo "Error: Failed to remove log directory '${LOG_DIR}'." >&2
            exit 1
        }
    else
        echo "Log directory '${LOG_DIR}' does not exist, skipping."
    fi

    echo "Uninstallation completed."
}

# Shows usage instructions
show_help() {
    echo "Usage: $0 {install|uninstall}"
    echo
    echo "Commands:"
    echo "  install     Install and enable the savt-client-worker service"
    echo "  uninstall   Stop and uninstall the savt-client-worker service"
    echo
}

# -------------------------------
# 4. Main function
# -------------------------------
main() {
    check_root
    check_systemd

    if [[ $# -eq 0 ]]; then
        echo "Error: Operation parameter (install | uninstall) is required." >&2
        show_help
        exit 1
    fi

    case "$1" in
        install)
            echo "Attempting to remove any previous installation ..."
            uninstall || true

            install_executable
            if [[ -z "${EXECUTABLE_PATH}" ]]; then
                echo "Error: No executable matching pattern '${EXECUTABLE_NAME_PATTERN}' found in ${INSTALL_DIR}." >&2
                exit 1
            fi

            install_config
            create_service_file
            reload_systemd
            enable_and_start_service
            show_service_status
            echo "Installation completed. ${SERVICE_NAME} service is registered and started."
            ;;
        uninstall)
            uninstall
            ;;
        *)
            echo "Error: Invalid parameter '$1'." >&2
            show_help
            exit 1
            ;;
    esac
}

# -------------------------------
# 5. Execute main function
# -------------------------------
main "$@"