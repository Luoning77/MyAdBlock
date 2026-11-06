#!/system/bin/sh
# Installer API shared by Magisk, KernelSU and APatch. No network or system settings changes here.
SKIPMOUNT=true
PROPFILE=false
POSTFSDATA=false
LATESTARTSERVICE=true
ui_print '- MyAdBlock：ARM64、安全旁路、离线 Miuix'
case "$ARCH" in arm64) ;; *) abort '只支持 ARM64；未修改网络或私人 DNS' ;; esac
[ "$API" -ge 29 ] || abort '需要 Android 10 或以上版本'
[ "$BOOTMODE" = true ] || abort '请在 Root 管理器 App 内安装，暂不支持 recovery'
[ -f "$MODPATH/bin/AdGuardHome" ] || abort '缺少官方 ARM64 内核'
[ -f "$MODPATH/webroot/myadblock.js" ] || abort '缺少已编译的离线 Miuix 资源'
[ -f "$MODPATH/lib/network-probe.jar" ] || abort '缺少网络信息辅助程序'
touch "$MODPATH/skip_mount"
set_perm_recursive "$MODPATH" 0 0 0755 0644
set_perm_recursive "$MODPATH/bin" 0 0 0700 0700
for script in customize.sh service.sh action.sh uninstall.sh emergency.sh; do
 set_perm "$MODPATH/$script" 0 0 0700
done
ui_print '- 账号 root / 密码 root；默认凭据公开，请在设置中修改'
ui_print '- 升级保留 /data/adb/myadblock 数据；重启后自动启用'
ui_print '- 首次有效接管会备份并关闭私人 DNS；未知 VPN/规则冲突时保持旁路'
ui_print '- Android 17 / KernelSU / Edge 为目标环境，当前尚未实机验证'
