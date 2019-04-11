#!/bin/sh

HOST_DIR=$1

items="MemTotal MemFree MemAvailable Buffers Cached"

for item in $items ; do
  val=$(grep -e ^$item: $HOST_DIR/proc/meminfo | awk '{print $2}')
  json="$json\"$item\":$val,"
done

# Output json
echo "{${json%,}}"

return 0
