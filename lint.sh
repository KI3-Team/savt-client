#!/bin/bash

set -e

MODULES=(
    #"savt-client-api"
    "savt-client-cli"
    "savt-client-worker"
    #"savt-client-gui"
)

CURRENT_DIR=$(pwd)

for module in "${MODULES[@]}"; do
    echo "linting $module"
    cd "$CURRENT_DIR/$module"
    golangci-lint run ./...
    echo "----------------------------------------"
done

cd "$CURRENT_DIR" 