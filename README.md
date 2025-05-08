# SAVT Client

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

SAVT Client is a high-performance network diagnostic tool for analyzing and optimizing network performance.

## Features

- Network Latency Testing
- Bandwidth Measurement
- Route Tracing
- Packet Analysis
- Real-time Monitoring
- Multi-platform Support

## Quick Start

### Installation

```bash
# Install using package manager
brew install savt-client  # macOS
apt install savt-client   # Ubuntu/Debian

# Or build from source
git clone https://github.com/your-username/savt-client.git
cd savt-client
make build
```

### Usage

```bash
# Basic usage
savt-client ping example.com

# Bandwidth test
savt-client speedtest

# Route tracing
savt-client trace example.com
```

## Documentation

- [User Guide](docs/user-guide/README.md)
- [API Documentation](docs/api/README.md)

## Feedback

If you encounter any issues or have feature suggestions while using the software, please provide feedback through [GitHub Issues](https://github.com/your-username/savt-client/issues).

## Security

If you discover a security vulnerability, please refer to our [Security Policy](SECURITY.md).

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

## Contact

- Feedback: [GitHub Issues](https://github.com/your-username/savt-client/issues)
- Email: [Project Email]
- Website: [Project Website]


![icon.ico](./icon.ico)


# SAVT Client Project

## Build Process

### 0. Clean Build Environment
```bash
rm -rf build/
```

### 1. Compilation Phase
```bash
# Compile savt-client-api
cd savt-client-api
go build -o ../build/savt-client-api

# Compile savt-client-cli
cd ../savt-client-cli
go build -o ../build/savt-client-cli
```

### 2. Packaging Phase
```bash
cd ..
./package.sh
```

Goal: Unified packaging logic, ultimately combining the build artifacts (binary files) of these two modules into a single software package.

Packaging format can be chosen based on the target platform:
- MacOS: Use pkgbuild tool to package into dmg installer
- Windows: Use InnoSetup tool to package into a single exe installer

### 3. Integration and Deployment
```bash
make deploy
```

### Package Operation Example
```bash
git checkout release/2.0.0
# a. Execute command on 2 mac arm64 and amd64 computers, manually ensure not to modify source code!!
make flow
# b. Execute command on 1 arm64 windows arm64 computer
make flow
# c. Execute command on 2 arm64/amd64 linux computers
make flow
```

