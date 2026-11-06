# 设计决策与兼容边界

2026-11-05。以 TASK_SPEC.md 为验收依据；目标真机 Android 17 / ARM64 / KernelSU / Microsoft Edge，当前开发机 macOS ARM64。

1. **控制层**：静态 Go Linux ARM64 二进制使用 Linux 系统调用，可在 Android Root 环境运行；真正的 Bionic/SELinux/Root 管理器兼容性仍需真机。AGH 为官方未修改 linux_arm64 0.107.80，一个子进程。无 32 位构建。
2. **网络安全**：只管理 NAT 的 MYADBLOCK_DNS 链与其精确 OUTPUT 跳转，不使用 DROP，不修改 PREROUTING、代理文件、系统 DNS 属性或飞行模式。OUTPUT 存在任何其他 NAT 规则时保守旁路。这可能对正常 OEM NAT 也降级，代价是覆盖率降低，避免猜测兼容。
3. **回退语义**：不使用 AGH 原生 fallback_dns；始终清空该列表，控制层只传实测健康公共 DoH，或仅传实际系统 DNS。默认探测每 5 秒调度，单个公共探测 3 秒，系统 UDP DNS 2 秒并发。NXDOMAIN 是合法响应，SERVFAIL/超时/TLS 故障不是。任一公共可用仍优先公共；全失败计数 3 次，恢复计数 3 次。等待回退期间先旁路，避免用户等坏 DoH 超时；这比硬维持 PUBLIC_DOH 更保守。
4. **网络来源**：通过 app_process + ConnectivityManager/LinkProperties 读取网络；不解析 dumpsys/net.dns 属性，不读取 SSID。活动网络为 VPN 时即使能找到底层 DNS，也不接管。底层网络选择有歧义、DNS 为虚拟/回环/无效地址、接口疑似 TUN 时旁路。getAllNetworks 为可编译但被弃用的 API，其 Root 调用权限及 OEM 实现需要实机证据。
5. **防循环**：AGH、bootstrap、健康探测均 root 出站，独立链第一条 UID0 RETURN。因此 root 应用的 DNS 不过滤，这是明确的覆盖限制。模块自身永远不会重定向自身公共探测/系统 DNS 回退。未知 VPN/TUN 或其他 NAT 路径保留。
6. **真实接管测试**：每个地址族首次加规则后，启动同一控制程序的短命探测子进程，进入后降为 UID/GID 9999（inet 组），查询文档保留目的地址的 TCP/UDP 53，必须收到 AGH 保留域 myadblock-health.invalid 的 192.0.2.173 canary 回答。该目的地址不提供真实 DNS 服务，成功证明 NAT 路径到达此核心；失败撤销该族。探测 UID 在部分 Android 上无法联网会保守降级。保留 canary rewrite，不影响其他用户 rewrite。
7. **暂停与 conntrack**：仅删除 NAT 规则不足以撤销既有连接映射。控制层通过 NETLINK_NETFILTER 获取连接表，仅删除“原目的端口 53 + 回包源为 loopback 当前随机 DNS 端口”的 TCP/UDP 元组，并携带内核 ID/zone 防止元组重用误删。无全局 flush。内核不支持受限 conntrack 操作时不启用接管。解析器有边界及选择性测试；真实 Android ctnetlink 权限、连接中断后的系统重试仍未测。
8. **故障隔离**：bridge 与控制器在同一进程的独立 HTTP goroutine，单个请求 panic 由 net/http 隔离。独立进程 watchdog 检查单调 uptime 租约，超过 15 秒撤销规则和本模块连接映射；系统调用有超时，因此该值不是绝对的断网时限。AGH 崩溃时控制器优先旁路，5 分钟内最多启动 3 次。网页关闭不影响守护。
9. **端口**：crypto/rand 选择 20000–59999，先绑定预留，AGH 启动释放预留后检查监听与 UDP/TCP 应答。交给 AGH 时仍有不可完全消除的跨进程竞态；碰撞则启动失败并保持旁路。每次启动服务会重新分配，包含每次冷启动；同一开机内守护重启端口也可能变化。端口只写 root 状态，YAML 在 AGH 停止时结构化修改，保留用户字段。
10. **认证**：root/root 严格保留；bridge 30 分钟 HttpOnly SameSite=Strict 会话、Host/Origin/Fetch-Metadata/CSRF 检查、固定 API、请求大小及登录限流。管理器 action 产生一分钟单次能力，在 URL fragment 中交换后立即清除，为该会话授予 5 分钟写权限，不写日志/网页存储。AGH 另有本机随机控制账号，密码只放 root 文件。AGH 默认 root/root API 仍可被恶意本机应用直接访问；必须更换密码才能消除此默认密码风险。
11. **配置**：仅支持 AGH schema 34，未知版本拒绝覆盖。每次受控写入先备份 last-good.yaml，再官方 --check-config，再原子替换；不在 AGH 运行时手改 YAML。便携导入只允许缓存、用户规则、有限日志/统计字段，拒绝用户/路径/监听/任意上游字段；DoH 在类型化设置 API 中改。它不是任意 AGH YAML 全量导入器。完整配置备份在 root 数据目录；Web 导出无管理凭据。
12. **更新**：订阅由控制器下载、验证非空/大小/格式，再通过受限本地文件路径让 AGH 载入；失败保留旧快照。自动检查每运行 24 小时，离线不阻塞开机。未实现 AGH 二进制在线替换或自动模块升级，更新需重新构建/刷包；使用固定版本与校验。每日调度在守护重启后重新计时，频繁重启设备时应手动检查。
13. **Miuix**：锁定 0.9.4、Kotlin 2.4.20、Compose 1.12.0，真实 Kotlin/JS target 构建通过。仍包含 Skiko WebAssembly 运行时，但不要求 Wasm GC。静态 JS/wasm/中文字体完全离线。低版本浏览器、Android Edge 的触控、无障碍、后台计时尚无真机证据；浏览器倒计时只是显示，恢复由守护单调时钟决定。
14. **只读未覆盖场景**：无 SNI/NFQUEUE/MITM、无 App 缓存清理、无校内域名自动分流、无 VPN 内全覆盖保证、无远程管理及遥测。不模拟支持 NFT-only；iptables/ip6tables 能力不足就旁路。

官方核验来源：
- https://github.com/AdguardTeam/AdGuardHome/blob/v0.107.80/openapi/openapi.yaml（保护 duration 单位毫秒，代码发送 0 表示无限，由守护计时）
- https://compose-miuix-ui.github.io/miuix/guide/multiplatform（JS target 支持）
- https://github.com/compose-miuix-ui/miuix/tree/39c40f99844227b853f0049a0933b1f3ae6c00ba
- https://topjohnwu.github.io/Magisk/guides.html
- https://developer.android.com/reference/android/net/LinkProperties
- https://github.com/torvalds/linux/blob/v6.12/include/uapi/linux/netfilter/nfnetlink_conntrack.h

私人 DNS 采用写前日志和成功读回确认。只对记录为 Applied 的字段进行条件恢复；进程在系统写入与确认落盘之间被杀时，保守保留当前值，可能需要用户手动恢复私人 DNS，以免认领其他组件后来写入的值。

AGH profile/update 不支持密码字段。密码修改通过撤销接管、受控停机、原子修改 root bcrypt、官方配置校验再启动实现；回归实际验证新密码可登录、旧密码失效，以及导入拒绝与升级规范化保留凭据。

静态 Go 的 DoH 探测与订阅下载都不使用系统默认 resolv.conf：显式并行查询全部 Bootstrap 的 A/AAAA，合并当前可靠网络 DNS，缓存地址最多60秒，公共 DoH 请求失败会清除缓存重查。避免 UDP connect 成功就误以为首个 Bootstrap 能应答；bootstrap 查询同样通过 UID0 绕开自身接管。测试涵盖首个不应答、第二个可用。

暂停恢复只依赖暂停记录本身的 boot ID，不依赖端口文件。端口文件丢失/损坏不会取消同次开机的暂停；暂停记录损坏时保守无限暂停，等待手动恢复。模拟 CLI 要求独立数据目录，拒绝混用设备真实或未知守护的数据。

当前系统 DNS 发现不支持带 scope 的 IPv6 link-local DNS 地址；若网络只提供此类服务器会保守旁路，不能宣称所有 IPv6-only 网络都覆盖。
