#!/bin/sh

PROC_NAME=$1
HOST_DIR=$2

rss_total=0
hwm_total=0
cpu_total=0
alive="false"
pid=0
proc_num=0
pid_list=""

# get ps c -C proc_name
procs=$(chroot $HOST_DIR ps c -C $PROC_NAME --no-headers | awk '{printf "%d|%s\n",$1,$5}')

for ps in $procs ; do
  # pid
  pid=$(echo $ps | cut -d'|' -f1)
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
  if [ "$cgroup_path" != "/" ]; then
    cgroup_path=$HOST_DIR/sys/fs/cgroup/cpu,cpuacct${cgroup_path}/cpuacct.usage
    cpu_usage_nano=$(cat $cgroup_path)
    if [ -e "/tmp/$PROC_NAME-$pid-cpu" ]; then
      cpu_usage_nano_prev=$(cat /tmp/$PROC_NAME-$pid-cpu)
      elapsed=$(($(date +%s) - $(date +%s -r /tmp/$PROC_NAME-$pid-cpu)))
      cpu_usage=$((($cpu_usage_nano - $cpu_usage_nano_prev) / 1000000 / $elapsed))
    else
      cpu_usage=0
    fi
    echo $cpu_usage_nano > /tmp/$PROC_NAME-$pid-cpu
    cpu_total=$(($cpu_total + $cpu_usage))
  fi
  # proc_num
  proc_num=$(($proc_num + 1))
  # alive
  alive="true"
done

# Output json
echo {\"alive\":$alive,\"pid\":$pid,\"cpu_msec\":$cpu_total,\"VmHWM\":$hwm_total,\"VmRSS\":$rss_total,\"proc_num\":$proc_num}

return 0
