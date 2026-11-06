#!/system/bin/sh
MODDIR=${0%/*}
umask 077
"$MODDIR/bin/myadblock" grant --module "$MODDIR" --data /data/adb/myadblock || {
 echo 'WebUI 尚未就绪。请查看 /data/adb/myadblock/service.log'
 echo "应急暂停：sh $MODDIR/emergency.sh"
 exit 1
}
