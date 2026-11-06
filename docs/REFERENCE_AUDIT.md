# 参考代码审计
2026-11-05，完整阅读 TASK_SPEC.md，检查 reference/Adguard-Home-For-Magisk-Mod 中安装、服务、操作、卸载与四个附加脚本。

仅参考通用模块入口结构、AGH 配置字段与离线快照的用途。实现独立编写，不复制脚本或私人 YAML。

| 参考行为 | 风险 | MyAdBlock 决策 |
|---|---|---|
| 标记 hosts 模块删除 | 破坏其他模块 | 禁止；报告冲突 |
| IPv6 53、TCP/UDP 853 全局 DROP | 断网、私人 DNS/代理冲突 | 禁止；只创建自身 NAT 链，不支持就放行 |
| 全局 AGH 重定向无出站豁免 | 系统 DNS 回退递归 | UID0 出站豁免，探测独立 |
| 改 Clash/Mihomo 配置/订阅 | 配置损坏 | 不写代理文件；VPN/已有 DNS 接管时旁路 |
| 切飞行模式 | 全网络中断 | 禁止 |
| App 缓存/IFW 删除、chattr 锁定 | 破坏应用、系统 | 禁止 |
| 持续覆写私人 DNS | 抢夺用户设置 | 一次写入、字段条件恢复 |
| 无上限重启/脚本自锁 | 崩溃循环、不可维护 | 限流重启、外部网络 watchdog |
| RANDOM 与 sed 广泛改 YAML | 端口冲突、配置丢失 | crypto/rand、实际绑定、YAML 结构化事务 |
