#!/system/bin/sh
MODDIR=${0%/*}
umask 077
# Stops only recorded, identity-checked children; restores only unchanged settings written by us.
"$MODDIR/bin/myadblock" uninstall --module "$MODDIR" --data /data/adb/myadblock || {
 echo '自动清理未完成；数据已保留。请使用应急命令并检查私人 DNS。'
 exit 1
}
