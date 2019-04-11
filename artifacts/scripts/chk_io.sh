#!/bin/sh

DISK_NAME=$1
HOST_DIR=$2

# disk IOs
read=0
write=0
sector=512
prev_file="/tmp/diskstats-$DISK_NAME"

stats=$(grep " $DISK_NAME " $HOST_DIR/proc/diskstats)

if [ -e $prev_file ]; then
  elapsed=$(($(date +%s) - $(date +%s -r $prev_file)))
  prev_stats=$(cat $prev_file)
  read=$((($(echo $stats | awk '{print $6}') - $(echo $prev_stats | awk '{print $6}')) * $sector / $elapsed))
  write=$((($(echo $stats | awk '{print $10}') - $(echo $prev_stats | awk '{print $10}')) * $sector / $elapsed))
fi

echo $stats > $prev_file

# Output json
echo {\"disk_name\":\"${DISK_NAME}\",\"ioread\":${read},\"iowrite\":${write}}

return 0
