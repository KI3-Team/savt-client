# 3.0.0 双栈测量、原始 IP 报文和共享选择

同一次测量生成 measurementId 和 measurementToken，在 IPv4、IPv6 创建请求中使用相同值。families 声明本轮地址族；isPublic 来自现有客户端共享设置。版本继续使用 3.0.0。

1. POST /api/session 创建单栈会话。返回 id 是单栈探测编号。
2. 首次 /api/session/{id}/next 发送真实 localSourceIp；后续 /next 继续携带。NAT 后客户端使用实际网卡源，不以服务端看到的公网源替代。
3. 服务端返回待发完整 IP 字节。客户端补链路头、发送、监听、等待，按服务端动作执行。
4. 下一次 /next 上传原始 receivedPackets、sendResults 和 localSourceIp，由服务端判断探测结果。没有 listen 的轮次不凭空产生接收包。
5. 每栈最后一次 /next 保存结果；全部声明地址族终态后自动汇总，并返回同一个最终 sessionId。新客户端不调用独立 finish。

measurementId 关联整个任务；result.id/result6.id 是单栈探测编号；result.sessionId/result6.sessionId 是 final 编号，供下游关联双栈记录。服务端持久化已完成部分，并在无活动 15 分钟后汇总已有结果。

缺失 isPublic 表示未知，服务端仍保留所有本地结果，只暂停下游上传。false 和 true 不互相转换，不能用缺失值默认公开。后续配置变更用于新的任务。

不再下发 routeTargets 或 capabilities。客户端请求和原始字节保留实际捕获信息，测量语义在服务端。IPv6 不执行 NAT 映射/过滤两轮。

worker 与 CLI 支持独立 --server、--server6、--data-dir。macOS 测量受 Surge 接管影响；不得通过脚本修改 Surge。用户需亲自关闭并明确确认后，才执行依赖物理路由的测量。

已知限制：SIT/HE 隧道上的 libpcap 注入可能丢失报文；当前同步保留已验证候选，不包含尚未实现的隧道修复。仅状态完成或 HTTP 200 不等同于探测成功，应结合服务端原始归档、normal 对照和最终判定。
