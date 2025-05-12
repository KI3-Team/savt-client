# SAVT Client Developer Manual

## 1. Environment Requirements

Before starting the build process, please ensure your system meets the following requirements:

### 1.1 System Requirements Table

| System | Architecture | Version | Golang Version | Dependencies |
|--------|--------------|---------|----------------|--------------|
| Windows | x64/ARM64 | Windows 10 or higher | Go 1.21+ | - [make](https://www.gnu.org/software/make/)<br>- [jq](https://jqlang.github.io/jq/)<br>- [InnoSetup](https://jrsoftware.org/isinfo.php)<br>- [Cygwin](https://www.cygwin.com/) |
| MacOS | Apple Silicon/Intel | macOS 10.15 or higher | Go 1.21+ | - [make](https://www.gnu.org/software/make/)<br>- [jq](https://jqlang.github.io/jq/)<br>- [create-dmg](https://github.com/create-dmg/create-dmg) |
| Linux | x86_64/aarch64 | Linux Kernel 5.4+ | Go 1.21+ | - [make](https://www.gnu.org/software/make/)<br>- [jq](https://jqlang.github.io/jq/) |

### 1.2 Quick Installation Guide

#### Windows
```bash
choco install make
choco install jq
# Install InnoSetup
# Download and install from https://jrsoftware.org/isdl.php
# Install Cygwin
# Download and install from https://www.cygwin.com/
```

#### MacOS
```bash
brew install make
brew install jq
brew install create-dmg
```

#### Linux
```bash
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install make jq
# CentOS/RHEL
sudo yum install make jq
```

## 2. Build Process

### 2.1. Environment Preparation

First, ensure you are in the project root directory and have installed all necessary dependencies.

### 2.2. Version and Environment Setup

```bash
# Set version (automatically generated from git tag and commit hash)
make version
# Set environment to test
make env-test
# Or set environment to production
make env-production
```

### 2.3. Build Process

If you need to execute individual steps, you can use the following commands:

```bash
# Clean build environment
make clean
# Build binary files for current platform
make build
# Package installation files for current platform
make package
```

Or, you can use a single command to execute the complete process:

```bash
make clean build package
```

### 2.4. Build Output

After the build is complete, you can find the following files in the `build` directory:

- MacOS: `.dmg` installation package
- Windows: `.exe` installation package
- Linux: `.tar.gz` archive


