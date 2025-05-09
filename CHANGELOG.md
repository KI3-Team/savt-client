# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- 支持 Windows ARM64 构建目标
- 新增 `make deploy_doc` 命令用于发布文档

### Fixed
- 修复打包流程在 macOS 上不兼容的问题

## [v1.2.0] - 2025-05-01

### Added
- 新增 `make release_doc` 用于文档版本化部署

### Changed
- 更新 `deploy` 逻辑，支持动态 ENV 选择

## [v1.1.0] - 2025-03-15

- 初始版本发布，包含 CLI、GUI、Worker 三组件支持
