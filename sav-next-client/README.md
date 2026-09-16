# sav-next-client

新一代（原语化重构）SAV 测量客户端 —— **独立 CLI、独立 Go 模块**，不依赖 `savt-client-worker` / `savt-client-api`。

## 与旧版客户端的关系

| | 旧版客户端 | 新版（本目录） |
|---|---|---|
| 组件 | `savt-client-gui` + `savt-client-worker` + `savt-client-cli` | 单一 CLI 可执行文件 |
| 通信 | gRPC（与本仓库 worker 通信） | HTTP 轮次协议（与 sav-server 控制器通信） |
| 测量逻辑 | 内置于 worker（outbound/inbound/tracefilter/traceroute 各自实现） | **不含测量语义**：由服务端下发原语动作（send / listen / wait），客户端只做执行 |
| 代码规模 | 10,000+ 行（不含 GUI） | ~740 行（客户端 + 共享库） |

新增测量类型只需改动服务端策略与判定，客户端零改动。

## 结构

```
sav-next-client/
├── cmd/client/main.go   # 原语解释器：会话循环、动作执行、ICMP匹配、结果上报
├── common/
│   ├── common.go        # 原语类型定义、HMAC 载荷编解码
│   └── pcapdev.go       # 网卡发现、包构造与 pcap 注入
└── go.mod
```

## 编译

```bash
go build -o bin/sav-next-client ./cmd/client
```

依赖：Go 1.22+、gopacket（cgo，需 libpcap 开发包）。

Windows 交叉编译（需 Npcap SDK）：

```bash
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
  CGO_CFLAGS="-I<npcap-sdk>/Include" \
  CGO_LDFLAGS="-L<npcap-sdk>/Lib/x64 -lwpcap -lws2_32 -liphlpapi" \
  go build -o bin/sav-next-client.exe ./cmd/client
```

macOS 需本机编译（cgo 无法跨平台交叉编译）。

## 运行

```bash
sudo ./bin/sav-next-client --server http://<控制器地址>:41452
```

- 需要 root / 管理员权限（pcap 注入 + 原始 socket）
- Windows 需安装 Npcap（勾选 WinPcap API-compatible Mode）
- 运行前请关闭 VPN / 代理 TUN 模式（否则出口与测量结果被污染）

输出：逐轮执行日志 + 最终结果 JSON（结果同时由服务端落盘）。

## 平台支持

Linux / macOS / Windows（arm64、amd64）均已实测通过。
