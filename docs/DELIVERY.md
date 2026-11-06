# 分阶段交付记录

日期：2026-11-05。任务书完整读取，目录首次只有 TASK_SPEC.md 和 reference/。未推送、未发布 Release、未连接或修改 Android 设备。

## P0 — 核心、安全安装与旁路

产物：module/ 五个标准脚本与应急入口、官方安装器；cmd/myadblock、internal/config / agh / network / engine；固定官方 ARM64 核心、三份离线规则缓存，随机端口和首次 YAML 生成。ZIP 包含离线资源和许可。

本机通过：ARM64 ELF 构建、脚本语法、ZIP 静态审计；真实 AGH 配置检查、无初始化 API、离线拦截、默认 root/root。故障规则回滚、独立链不影响外部规则、私人 DNS 字段条件恢复由单元测试覆盖。

未验证：任何 Root 管理器真机首装/卸载、Android DNS NAT 和实际断网恢复。当前 ZIP 属于可安装结构的待实机验收包，不能宣称 P0 真机验收完成。

下一步：优先在用户 Android 17 / KernelSU / Edge 设备完成首装、旁路与卸载；验证失败先保持安全旁路，不强行加规则。

## P1 — 动态回退、完整暂停与代理兼容

产物：Android NetworkProbe.java + DEX；LinkProperties 发现、四态状态机、三次防抖、按健康选择上游、两族能力与 canary 验证、精准 conntrack 清理、独立 watchdog、定时/无限暂停、启动清除暂停、代理检测。

本机通过：单公共上游优先、三次切换、全坏旁路、暂停不被自愈覆盖、无限暂停/到期状态、NXDOMAIN 合法、IPv6 失败回滚、conntrack 解析/精确目标选择；实际 AGH 暂停/恢复与 UDP/TCP canary 在 macOS 集成测试覆盖。

未验证：真实 Android ctnetlink 权限与删除语义、非 root 探测 UID 的路由能力、Wi-Fi/移动切换、IPv6-only、校园 Portal、各代理模式。已提供 CI 的独立 Linux network namespace NAT/conntrack 集成测试，当前环境尚未执行该 Linux 专项。

下一步：先验证目标设备 LinkProperties 与 netlink 权限，再跑完整网络/代理矩阵。能力缺失会明确旁路。

## P2 — 正式 Miuix 与安全 bridge

产物：webui/ Kotlin/JS + Miuix 0.9.4 四页、底部暂停弹窗、搜索/分页日志、黑白名单、预设/订阅管理、类型化设置、备份/恢复、密码修改、备用官方 UI；internal/bridge 会话/授权边界。

本机通过：真实 Kotlin/Compose 编译和离线 Webpack 发行构建；应用内浏览器登录、中文字体、390×844 页面观察和导航；bridge Host/Origin/未认证/无短期授权/无 shell 端点测试。截图是明确标为“模拟运行”的实际页面，不是设计图。

未验证：Android Edge、不同 WebView/老版本浏览器、Android 触控/键盘/无障碍完整行为。桌面 Canvas 语义点击和实际坐标点击存在浏览器自动化差异，不能当真机触控结论。

下一步：在目标 Edge 完成登录、短期授权、全部暂停选项、倒计时、后台再打开、配置导入、过滤模式和错误状态的交互验收。

## P3 — 构建、审计与开源材料

产物：version.properties、tools/fetch.py / build.py / audit.py、tests/test.sh、Go tests、锁文件与 SHA-256、NOTICE / licenses、双语 README、CI、个人构建、恢复文档和完整测试矩阵。

本机通过：核心官方校验和一致、公开打包白名单、禁止危险脚本审计、完整发行构建、重复打包哈希比对（结果见 FINAL_VERIFICATION.md）。代码与产物已交到本地，没有创建远程仓库或发布。

未完成的发行验收：Root / 网络 / 代理 / Android 浏览器真机矩阵；CI 工作流仅写好并静态检查，未远程运行；CI Actions 已经官方标签解析锁到 commit SHA，工作流尚待远程执行。核心/模块在线升级未实现；采用明确的手动刷包及原配置回滚，不能称为在线内核升级器。

下一步：带真机证据补齐矩阵、跑一次受控 CI、审计准备公开的源码树，再由用户决定正式发行。

## 范围收紧与限制

- 完整官方 YAML 导入收紧为便携白名单，防止路径与监听逃逸；上游通过类型化 DNS 设置修改。
- 未知 VPN/TUN、任意其他 OUTPUT NAT、缺少 conntrack 或不能验证 canary 时安全降级；不承诺全场景兼容。
- 每次服务重启也会变随机端口；用户配置保留。跨进程端口交接仍有竞态，失败旁路。
- 每运行 24 小时检查订阅，守护重启后重新计时，频繁重启可手动更新。
- UI 统计按本地今日小时向 AGH 请求，不伪造网络覆盖或统计；前端失联明确显示未知。

完整技术理由见 DESIGN_DECISIONS.md；不能把以上未验证内容写成“已支持且已验证”。

仓库安排（2026-11-06）：按用户最新决定改用独立仓库 https://github.com/Luoning77/MyAdBlock ，不再向参考项目的 fork 提交。Go 模块路径与新仓库一致，中英文 README 保留对原项目与参考分支的致敬。首次公开提交只包含 MyAdBlock 当前源码，不继承参考项目历史；旧历史保留在本地 archive/legacy-reference 分支，不推送。Android 真机待验收范围不变。
