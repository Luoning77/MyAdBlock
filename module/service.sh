#!/system/bin/sh
MODDIR=${0%/*}
umask 077
[ -e "$MODDIR/disable" ] && exit 0
# late-start runs asynchronously. No boot-completed infinite wait.
mkdir -p /data/adb/myadblock || exit 1
chmod 0700 /data/adb/myadblock
LOG=/data/adb/myadblock/service.log
[ -f "$LOG" ] && [ "$(wc -c < "$LOG")" -gt 65536 ] && mv -f "$LOG" "$LOG.1"
"$MODDIR/bin/myadblock" daemon --module "$MODDIR" --data /data/adb/myadblock >> "$LOG" 2>&1 &
