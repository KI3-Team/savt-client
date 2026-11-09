# 3.0.0 源码同步（2026-10-14）

- worker、CLI 共用原语执行引擎，GUI 保留现有 gRPC 接口。
- 客户端上报真实 localSourceIp，发送服务端生成的 IP 字节，回传原始 receivedPackets。
- 同一轮双栈 measurementId 关联、/next 自动结束与统一最终 sessionId。
- 共享配置明确传递 isPublic；版本继续使用 3.0.0。
- 更新器客户端避免在以太网环境永久等待 wifi_only 条件。

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [SAV-T-Client](https://github.com/KI3-Team/savt-client).

## [2.1.0] - 2025-04-12

### Changed
- Reduced the number of background measurements during program startup
- Added a "Status Indicator" column to the measurement tasks
- Fixed the issue where the client measurement progress displayed abnormally when a measurement failed

## [2.0.0] - 2025-01-23

### Changed
- Renamed the client software package from SAV-T to SAVT-Client according to the SAV-T specification

### Added
- Added measurement history display module
- Added scheduler module for automatic measurements
- Added traceroute and tracefilter measurement modules

## [1.2.1] - 2024-07-19

### Fixed
- Fixed GUI bug

## [1.2.0] - 2024-07-15

### Added
- Added the function to allow users to choose whether to make test results public or not

### Fixed
- Fixed GUI display bug

## [1.1.0] - 2024-07-08

### Added
- First public release, with the following functions:
  - IPv4 and IPv6 outbound and inbound SAV configuration detection
  - Adjacent netblock scanning to determine filtering granularity
