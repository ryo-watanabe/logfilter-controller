#!/bin/sh

HOST_DIR=$1

user=0
nice=0
syst=0
idle=0
iowa=0
per_val=1000

stats=$(head -1 $HOST_DIR/proc/stat)

if [ -e "/tmp/stats" ]; then
  total=0
  for val in $stats ; do
    if [ "$val" = "cpu" ] ; then
      continue
    fi
    total=$(($total + $val))
  done
  prev_total=0
  prev_stats=$(cat /tmp/stats)
  for val in $prev_stats ;  do
    if [ "$val" = "cpu" ] ; then
      continue
    fi
    prev_total=$(($prev_total + $val))
  done

  total=$(($total - $prev_total))

  user=$((($(echo $stats | awk '{print $2}') - $(echo $prev_stats | awk '{print $2}')) * $per_val / $total))
  nice=$((($(echo $stats | awk '{print $3}') - $(echo $prev_stats | awk '{print $3}')) * $per_val / $total))
  syst=$((($(echo $stats | awk '{print $4}') - $(echo $prev_stats | awk '{print $4}')) * $per_val / $total))
  idle=$((($(echo $stats | awk '{print $5}') - $(echo $prev_stats | awk '{print $5}')) * $per_val / $total))
  iowa=$((($(echo $stats | awk '{print $6}') - $(echo $prev_stats | awk '{print $6}')) * $per_val / $total))
fi

echo $stats > /tmp/stats

# Output json
echo {\"user\":$user,\"nice\":$nice,\"system\":$syst,\"idle\":$idle,\"iowait\":$iowa}

return 0
