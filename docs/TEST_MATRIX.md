# 测试矩阵与证据

本文件记录 2026-11-05 实际证据，未执行项一律不视为通过。

| 测试 | 本机结果 | 证据 / 限制 |
|---|---|---|
| Go 单元测试与 race | 已通过，最终日志随交付更新 | tests/test.sh；状态机/暂停/单 DoH/故障旁路/规则幂等与回滚/私人 DNS 条件恢复/配置白名单/bridge 边界 |
| 固定 AGH 0.107.80 本机集成 | 已通过 | macOS ARM64 官方核心，真实 DNS UDP、本地 mock 上游；跳过初始化、离线过滤、保护 API、双预设、用户规则、密码隔离、非法导入 |
| TCP/UDP canary、conntrack 精确选择 | 解析与选择性单元测试；Android 未测 | 不冒充真实 NAT/ctnetlink 测试 |
| ARM64 控制器交叉编译 | 已通过 | GOOS=linux GOARCH=arm64 CGO_ENABLED=0，包审计检查 ELF64 AArch64 |
| Java 辅助程序 / DEX | 已编译 | Android API 37 SDK，min API 29；getAllNetworks 弃用警告，不是编译失败 |
| Miuix 0.9.4 Kotlin/JS | 已构建 | Gradle jsBrowserDistribution；JS、Skiko wasm、中文字体全离线 |
| 浏览器 UI | macOS 应用内浏览器观察 | 实际登录、首页/设置/规则、390×844 响应布局与中文字形；模拟模式显示未接管，不能等同 Edge Android 真机 |
| ARM64 ZIP 静态审计 | 已通过 | tools/audit.py；架构/必需入口/依赖哈希/禁止危险操作/无源码映射与私密文件 |
| 确定性 ZIP | 最终构建后比对 | 固定顺序/时间/权限、无二进制 build ID；跨 OS/JDK 压缩器位级相同不作保证 |
| Root 框架 | 全部未真机验证 | Magisk / KernelSU / APatch：首装、重启、action、升级、卸载 |
| Android 17 + KernelSU + Edge | 未真机验证 | 用户目标设备；本环境无连接设备授权，不自动改设备 |
| Android 10–16 / 其他浏览器 | 未验证 | API 级别门槛不是兼容性证明；老浏览器可能不能运行 Canvas/wasm |
| Wi-Fi/移动/双栈/IPv6-only/Portal | 未真机验证 | 含 Wi-Fi↔移动切换、断网恢复、校园认证与暂停 |
| 真实 DoH 六种健康组合 | 状态逻辑已测；真实网络未测 | 阿里/CF 单坏、双坏系统好、全坏、恢复；NXDOMAIN 单测通过 |
| FlClash VPN/TUN / Mihomo Fake-IP / redir-host / Box | 未真机验证 | 代码检测与保守旁路，不写代理配置；不能宣称全兼容 |
| AGH/bridge/watchdog 崩溃 | 组件逻辑已测；真机租约清理未测 | Android SIGKILL、内核缺少 NAT/nft-only、ctnetlink 权限、SELinux、Root 被撤销需要实测 |
| 端口冲突与开机暂停重置 | 分配/状态代码 + 单测；真机未测 | 跨进程 bind 竞态会安全旁路，不能宣称无竞态 |
| 跨站 / 无会话 / Host / Origin / CSRF / 任意 shell | 已通过本机单测 | 浏览器实际跨站利用场景仍需设备验收；AGH 默认凭据残余风险单列 |

## 真机验收建议

1. 先备份设备，保存旧模块 ZIP 和配置，在 KernelSU 刷入本包、重启，action 打开 Edge；确认根目录状态无“初始化向导”。
2. 对照页面核心运行、两族覆盖、当前上游分别检查。出现旁路时先读取 root 的 status.json 与日志，不强制添加全局规则。
3. 对实际可达广告域名查询，核对 AGH 原因；接着停止 AGH/控制守护，检查仅 MYADBLOCK_DNS 链撤销、网络恢复原路径。
4. 逐一测试所有暂停选项、暂停时切网/代理切换、到期失败保持旁路、无限暂停及重启恢复；检查旧 UDP/TCP 映射已移除。
5. 在可控网络分别阻断两个 DoH，验证至少一好仍公共、两坏才系统、恢复 3 次切回、全坏旁路；合法 NXDOMAIN 不计失败。
6. 对每个 Root 框架执行安装/升级/卸载，确保别的模块链、代理 YAML、App 数据不变化；私人 DNS 后来被用户修改时不覆盖。
7. 记录设备型号、系统构建、内核、管理器与浏览器版本、操作、观察、时间和脱敏日志，才能把矩阵某项改成“实机通过”。
