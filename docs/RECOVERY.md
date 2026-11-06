# 故障恢复与卸载

网页不可用时在 Root 终端执行（默认安装模块 ID 为 myadblock，脚本通过自身路径定位）：

```sh
sh /data/adb/modules/myadblock/emergency.sh
```

命令先写应急暂停标记，再只移除本模块 NAT 跳转/链及自身 DNS conntrack，通知守护无限暂停过滤。即使浏览器坏了也可执行；不会删除其他模块配置。也可以在管理器停用模块并重启，内核旧规则会随重启消失。应急暂停同样在下次冷开机清除。

```sh
# 永久退出：停止自己的进程、撤销接管、条件恢复私人 DNS；数据保留
/data/adb/modules/myadblock/bin/myadblock stop --module /data/adb/modules/myadblock
# 查看状态（root 文件，无密码）
/data/adb/modules/myadblock/bin/myadblock status
```

临时暂停保持私人 DNS 关闭；stop/卸载恢复自己实际改过且仍为 off 的字段。原 private_dns_specifier 被备份但不修改；若用户后来设置了新模式，不覆盖。系统无法读取/写入时应检查 Root/SELinux，不关闭整机 SELinux。

数据目录 /data/adb/myadblock 为 0700，文件 0600。AdGuardHome.yaml 包含凭据哈希，management.secret 是设备随机控制密码；不要分享这些文件或整个目录。诊断优先分享脱敏状态与 service.log，后者限制 64 KiB 并保留一份轮转。程序不打印认证材料或查询域名；默认 DNS 查询日志只在 AGH 内存中。

候选配置先经官方 --check-config，成功后将旧配置原子保存到 backups/last-good.yaml，再替换。需要手动回滚时先 stop，再将该备份复制到 AdGuardHome.yaml，然后重启服务；必须保证备份属于当前锁定 AGH schema 34。备份可能含旧凭据，回滚后确认密码。未知 schema 不自动强行迁移，保留旧数据并旁路。

核心与模块不在线自动替换。手动升级前保留旧 ZIP 与完整 root 数据备份；新版失败时停用、刷回旧 ZIP，再恢复匹配该 AGH 版本的 YAML。不要在 AGH 正运行时直接编辑 YAML，否则可能被其持久化覆盖。

卸载由管理器触发 uninstall.sh，只停止通过 PID + 命令行身份核对的本模块进程；撤销自己的规则、条件恢复私人 DNS，验证所有权标记后清理自身数据。权限不足/撤销失败时保留数据及诊断提示，不能宣称清理成功。没有备份且设备无法开机时，应通过 Root 管理器安全模式停用模块；本项目不提供修改其他 App 或系统目录的恢复脚本。
