#!/system/bin/sh
MODDIR=${0%/*}
exec "$MODDIR/bin/myadblock" emergency --module "$MODDIR" --data /data/adb/myadblock
