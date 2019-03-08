#!/bin/sh

PROC_NAME=$1
INTERVAL_SEC=$2
HOST_DIR=$3
CACHE_DIR=$4

rss_total=0
hwm_total=0
cpu_total=0
alive="false"
pid=0
proc_num=0
pid_list=""

if [ -e "$CACHE_DIR/$PROC_NAME-pid_list" ]; then
  # Read PIDs from file if available
  pid_list=$(cat $CACHE_DIR/$PROC_NAME-pid_list)
  for pid in $pid_list ; do
    if [ -e "$HOST_DIR/proc/$pid/status" ]; then
      proc_name=$(grep Name: $HOST_DIR/proc/$pid/status | awk '{print $2}')
      # If process name not match, remove pid-list
      if [ "$proc_name" != "$PROC_NAME" ]; then
        rm -f $CACHE_DIR/$PROC_NAME-pid_list
        pid_list=""
        break
      fi
    else
      # If process not found, remove pid-list
      rm -f $CACHE_DIR/$PROC_NAME-pid_list
      pid_list=""
      break
    fi
  done
else
  # If pid-list not found, create one
  for i in $HOST_DIR/proc/* ; do
    if [ -e "$i/status" ]; then
      proc_name=$(grep Name: $i/status | awk '{print $2}')
      if [ "$proc_name" = "$PROC_NAME" ]; then
        pid=$(grep ^Pid: $i/status | awk '{print $2}')
        echo "$pid" >> $CACHE_DIR/$PROC_NAME-pid_list
      fi
    fi
  done
  # Then read the list
  if [ -e "$CACHE_DIR/$PROC_NAME-pid_list" ]; then
    pid_list=$(cat $CACHE_DIR/$PROC_NAME-pid_list)
  fi
fi

# Summing memory and cpu usage
for pid in $pid_list ; do
  # memory
  hwm=$(grep VmHWM $HOST_DIR/proc/$pid/status | awk '{print $2}')
  rss=$(grep VmRSS $HOST_DIR/proc/$pid/status | awk '{print $2}')
  if [ "$hwm" != "" ]; then
    hwm_total=$(($hwm_total + $hwm))
  fi
  if [ "$rss" != "" ]; then
    rss_total=$(($rss_total + $rss))
  fi
  # cpu
  cgroup_path=$(grep cpuacct $HOST_DIR/proc/$pid/cgroup | cut -d':' -f3)
  if [ "$cgroup_path" != "" ]; then
    cgroup_path=$HOST_DIR/sys/fs/cgroup/cpu,cpuacct${cgroup_path}/cpuacct.usage
    cpu_usage_nano=$(cat $cgroup_path)
    if [ -e "$CACHE_DIR/$PROC_NAME-$pid-cpu" ]; then
      cpu_usage_nano_prev=$(cat $CACHE_DIR/$PROC_NAME-$pid-cpu)
      cpu_usage=$((($cpu_usage_nano - $cpu_usage_nano_prev) / 1000000 / $INTERVAL_SEC))
    else
      cpu_usage=0
    fi
    echo $cpu_usage_nano > $CACHE_DIR/$PROC_NAME-$pid-cpu
    cpu_total=$(($cpu_total + $cpu_usage))
  fi
  # proc_info
  proc_num=$(($proc_num + 1))
  alive="true"
done

# Output json
echo {\"alive\":$alive,\"pid\":$pid,\"cpu_msec\":$cpu_total,\"VmHWM\":$hwm_total,\"VmRSS\":$rss_total,\"proc_num\":$proc_num}

return 0
