<h1 align="center">
  <a href="https://github.com/dec0dOS/amazing-github-template">
    <img src="icon.ico" alt="Logo" width="250" height="250">
  </a>
</h1>

<div align="center">
  <br />
  <br />

  <a href="https://github.com/KI3-Team/savt-client/issues/new?assignees=&labels=bug&template=bug_report.md">Report a Bug</a>
  ·
  <a href="https://github.com/KI3-Team/savt-client/issues/new?assignees=&labels=enhancement&template=feature_request.md">Request a Feature</a>
  .
  <a href="https://ki3.org.cn/#/contact">Contact Us</a>
</div>

<div align="center">
<br />

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
<a href="https://github.com/KI3-Team/savt-client/stargazers"><img alt="GitHub stars" src="https://img.shields.io/github/stars/KI3-Team/savt-client?style=social"></a>

<a href="https://github.com/KI3-Team/savt-client/releases"><img alt="GitHub release (latest by date)" src="https://img.shields.io/github/v/release/KI3-Team/savt-client"></a>

</div>


## About

SAV-T client is an active measurement tool which attempts to send and receive a series of spoofed UDP packets to/from servers distributed throughout the world. We use SAV-T clients to test a network’s ability of outbound or inbound spoofing periodically.

**By participating and running SAV-T**, you help build a global dataset of source address validation capabilities, contributing to Internet security research and infrastructure protection. The collected results are anonymously visualized at [KI3 SAV-T Results Dashboard](https://ki3.org.cn/#/sav?sub=savTest&children=recentTestResults).



## Quick Start

### User Guide

- [User Guide](https://ki3.org.cn/#/sav?sub=savTest&children=clientDownload)

### Developer Guide

First, ensure you are in the project root directory and have installed all necessary dependencies：

| System | Architecture | Version | Golang Version | Dependencies |
|--------|--------------|---------|----------------|--------------|
| Windows | x64/ARM64 | Windows 10 or higher | Go 1.21+ | - [make](https://www.gnu.org/software/make/)<br>- [jq](https://jqlang.github.io/jq/)<br>- [InnoSetup](https://jrsoftware.org/isinfo.php)<br>- [Cygwin](https://www.cygwin.com/) |
| MacOS | Apple Silicon/Intel | macOS 10.15 or higher | Go 1.21+ | - [make](https://www.gnu.org/software/make/)<br>- [jq](https://jqlang.github.io/jq/)<br>- [create-dmg](https://github.com/create-dmg/create-dmg) |
| Linux | x86_64/aarch64 | Linux Kernel 5.4+ | Go 1.21+ | - [make](https://www.gnu.org/software/make/)<br>- [jq](https://jqlang.github.io/jq/) |

In most cases, the following commands are sufficient to build and install SAV-T-Client:
```bash
# Set version and set environment to test
make version env-test
# Use a single command to execute the complete process
make flow
```

After the build is complete, you can find the following files in the `build` directory:

- MacOS: `.dmg` installation package
- Windows: `.exe` installation package
- Linux: `.tar.gz` archive




## Contact
- Website: [https://ki3.org.cn/#/contact](https://ki3.org.cn/#/contact)
- Email: [ki3contact@163.com](mailto:ki3contact@163.com)



