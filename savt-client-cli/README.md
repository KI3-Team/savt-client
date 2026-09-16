# savt-client-cli

SAV 测量客户端（新一代 · **原语化重构版**）。

> 本目录原为基于 gRPC 的旧版 CLI（配合 `savt-client-worker`）。2026-09 起由原语化重构版替代，旧版代码见 git 历史（`git log -- savt-client-cli`）。

## 架构

客户端是**原语解释器**，不含任何测量语义：

- 从控制器（sav-server）获取轮次动作，执行 `send` / `listen` / `wait` 三类原语
- 测量类型（入向/出向/NAT画像/路径等）全部由服务端策略（strategy.json）下发
- 新增测量类型时客户端**零改动**

与服务端仓库配套：`ki3/sav/sav-server`（控制器）+ `ki3/sav/sav-inbound`（spoofer）。

## 结构

```
savt-client-cli/
├── main.go          # 原语解释器：会话循环、动作执行、ICMP匹配、结果上报
├── common/
│   ├── common.go    # 原语类型定义、HMAC 载荷编解码
│   └── pcapdev.go   # 网卡发现、包构造与 pcap 注入
└── go.mod
```

## 编译

```bash
go build .
```

依赖：Go 1.22+、gopacket（cgo，需 libpcap 开发包）。

Windows 交叉编译（需 Npcap SDK）：

```bash
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
  CGO_CFLAGS="-I<npcap-sdk>/Include" \
  CGO_LDFLAGS="-L<npcap-sdk>/Lib/x64 -lwpcap -lws2_32 -liphlpapi" \
  go build -o savt-client-cli.exe .
```

macOS 需本机编译（cgo 无法跨平台交叉编译）。

## 运行

```bash
sudo ./savt-client-cli --server http://<控制器地址>:41452
```

- 需要 root / 管理员权限（pcap 注入 + 原始 socket）
- Windows 需安装 Npcap（勾选 WinPcap API-compatible Mode）
- 运行前请关闭 VPN / 代理 TUN 模式（否则出口与测量结果被污染）

输出：逐轮执行日志 + 最终结果 JSON（结果同时由服务端落盘）。

## 平台支持

Linux / macOS / Windows（amd64、arm64）均已实测通过。
