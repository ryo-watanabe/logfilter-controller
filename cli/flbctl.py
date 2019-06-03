#!/usr/bin/env python3
#-*- coding: utf-8 -*-
# flbctl - python script

import argparse
import json
import datetime
import time
import subprocess
import sys, tempfile, os

GREEN = '\033[92m'
YELLOW = '\033[93m'
RED = '\033[91m'
ENDC = '\033[0m'

def localCommand(com):
    val = ""
    err = ""
    ret = 0
    try:
        out = subprocess.run(com.split(" "), check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    except subprocess.CalledProcessError as e:
        ret = e.returncode
        if e.stderr:
            err = e.stderr.decode("utf-8")
        return (ret, err)
    val = out.stdout.decode("utf-8")
    if out.stderr:
        err = out.stderr.decode("utf-8")
    return (ret, val + err)

def spacer(str, max):
    spaced = str
    while len(spaced) <= max:
        spaced += " "
    return spaced

def name_max(l, k0, k1, k2, lb):
    max = len(k0)
    for item in l:
        if not "labels" in item["metadata"] or not lb in item["metadata"]["labels"]:
            continue
        if not k2 in item[k1]:
            continue
        length = len(item[k1][k2])
        if length > max:
            max = length
    return max

def table(l,keys,lb):
    max = {}
    for ks in keys:
        max[ks[0]] = name_max(l, ks[0], ks[1], ks[2], lb) + 3
    table = ""
    for ks in keys:
        table += spacer(ks[0], max[ks[0]])
    table += "\n"
    for item in l:
        if not "labels" in item["metadata"] or not lb in item["metadata"]["labels"]:
            continue
        if item["metadata"]["labels"][lb] == "disabled":
            table += YELLOW
        for ks in keys:
            if not ks[2] in item[ks[1]]:
                table += spacer("N/A", max[ks[0]])
            else:
                table += spacer(item[ks[1]][ks[2]], max[ks[0]])
        if item["metadata"]["labels"][lb] == "disabled":
            table += "   DISABLED" + ENDC
        table += "\n"
    return table

def get_yaml(args):
    (ret, val) = localCommand("kubectl get cm " + args.name[0] + " -o yaml -n " + args.namespace)
    yaml = ""
    if ret == 0:
        lines = val.splitlines()
        for line in lines:
            if line.find("annotations:") >= 0 \
                or line.find("last-applied-configuration:") >= 0 \
                or line.find("\"apiVersion\"") >= 0 \
                or line.find("creationTimestamp:") >= 0 \
                or line.find("resourceVersion:") >= 0 \
                or line.find("selfLink:") >= 0 \
                or line.find("uid:") >= 0:
                continue
            yaml += line + "\n"
        return (ret, yaml)
    else:
        return (ret, val)

def get_data(args):
    if len(args.name) == 0:
        return (1, "Name not supplied.")
    (ret, val) = localCommand("kubectl get cm -n " + args.namespace + " -o json " + args.name[0])
    if ret == 0:
        return (0, json.loads(val))
    else:
        return (ret, val)

def get(args,keys,label):
    if len(args.name) > 0:
        (ret, val) = get_yaml(args)
        print (val, end="")
    else:
        (ret, val) = localCommand("kubectl get cm -o json -n " + args.namespace)
        if ret == 0:
            data = json.loads(val)
            t = table(data["items"], keys, label)
            print (t, end="")
        else:
            print (val, end="")

def newyaml(args,keys,label):
    yml = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n"
    yml += "  labels:\n    " + label + ': "true"\n'
    yml += "  name: " + args.name[0] + '\n'
    yml += "  namespace: " + args.namespace + "\ndata:\n"
    for k in keys:
        if k[2] == "name":
            continue
        yml += "  " + k[2] + ": " + k[3] + "\n"
    return yml

def keylabelchk(yml,keys,label):
    if yml.find(label) < 0:
        return False
    for k in keys:
        if yml.find("  " + k[2] + ": ") < 0:
            return False
    return True

log_help = """Subject 'log' ConfigMap data:

  log_kind      : k8s_pod_log       ... Tailing logs and add kubernetes.* by kubernetes_metadata_filter.
                : rke_container_log ... Tailing logs and add container_name by its file name.
                : syslog            ... Tailing logs and add log_name by file path.
  path          : Log path to tail. Currently mounted pathes are /var/log and /var/lib/rancher/rke/log only.
  tag           : Fluent-bit tags, also passed to elasticsearch as '_flb-key'. Asterisk will be replaced by log path.

"""
logfilter_help = """Subject 'logfilter' ConfigMap data:

  log_kind      : pod_log       ... Corresponding to log input k8s_pod_log.
                : container_log ... Corresponding to log input rke_container_log.
                : system_log    ... Corresponding to log input syslog.
  log_name      : Set container name for pod_log and container_log. Set log file path for system_log.
  message       : String to ignore/drop.
                  - Set '@all' to ignore/drop whole lines.
                  - Set '@startwith:XXXX' to ignore/drop lines start with XXXX.
                  - Use lua language string format. Hyphens '-' must be typed as '%-'.
  action        : ignore ... Add 'ignore_alerts' element and send it to elasticsearch.
                : drop   ... Do not send it to elasticsearch.

"""
proc_help = """Subject 'proc' ConfigMap data:

  proc_names    : Comma separated process names. Use true command names same as 'ps auxc' output.
  interval_sec  : Monitoring interval in seconds. Set number as a string like '60'.
  node_group    : Set a DaemonSet on which node the process will alive by nodegroup name .
  tag           : Fluent-bit tags, also passed to elasticsearch as '_flb-key'. Asterisk will be replaced by the proccess name.

"""
metric_help = """Subject 'metric' ConfigMap data:

  metric_kind   : pod  ... Get pod metrics from /apis/metrics.k8s.io/v1beta1/pods.
                : node ... Get node metrics from /apis/metrics.k8s.io/v1beta1/nodes.
  interval_sec  : Monitoring interval in seconds. Set number as a string like '60'.
  tag           : Fluent-bit tags, also passed to elasticsearch as '_flb-key'.

"""
output_help = """Subject 'output' ConfigMap data:

  host          : Elasticsearch host.
  port          : Elasticsearch port.
  index_prefix  : Cluster identifier in elasticsearch indeces. Index will be like '{index_prefix}-2019.01.01'
  match         : Match tag to send. Set '*' to send all to this output.

"""
nodegroup_help = """Subject 'nodegroup' ConfigMap data:

  node_selector : Comma separated NodeSelectors for the DaemonSet set as 'node-role.kubernetes.io/[node_selector]=true'.
  tolerations   : Comma separated Tolerations for the DaemonSet.
                  - The value 'etcd' is set as effect=NoExecute,key=node-role.kubernetes.io/etcd,value="true".
                  - The value 'controlplane' is set as effect=NoSchedule,key=node-role.kubernetes.io/controlplane,value="true".

"""
osmon_help = """Subject 'osmon' ConfigMap data:

  cpu_interval_sec        : Monitoring cpu interval in seconds. Set number as a string like '60'.
  cpu_tag                 : Fluent-bit tags, also passed to elasticsearch as '_flb-key'.
  memory_interval_sec     : Monitoring memory interval in seconds. Set number as a string like '60'.
  memory_tag              : Fluent-bit tags, also passed to elasticsearch as '_flb-key'.
  filesystem_df_dir       : Set directory name.
  filesystem_interval_sec : Monitoring filesystem interval in seconds. Set number as a string like '60'.
  filesystem_tag          : Fluent-bit tags, also passed to elasticsearch as '_flb-key'.
  io_diskname             : Set device name
  io_interval_sec         : Monitoring io interval in seconds. Set number as a string like '60'.
  io_tag                  : Fluent-bit tags, also passed to elasticsearch as '_flb-key'.

"""
app_help = """Subject 'app' ConfigMap data:

  app_kinds        : Comma separated k8s app kinds to monitor. To set all type 'deployments,daemonsets,statefulsets'.
  interval_sec     : Monitoring in seconds. Set number as a string like '60'.
  tag              : Fluent-bit tags, also passed to elasticsearch as '_flb-key'. Asterisk will be replaced by app kinds.

"""

def edit_help(args):
    if args.subject == "log":
        print (log_help, end="")
    if args.subject == "logfilter":
        print (logfilter_help, end="")
    if args.subject == "proc":
        print (proc_help, end="")
    if args.subject == "metric":
        print (metric_help, end="")
    if args.subject == "output":
        print (output_help, end="")
    if args.subject == "nodegroup":
        print (nodegroup_help, end="")
    if args.subject == "osmon":
        print (osmon_help, end="")
    if args.subject == "app":
        print (app_help, end="")

def edit(args,keys,label,create):
    if len(args.name) > 0:
        if args.name[0] == "help":
            edit_help(args)
            return
        if create:
            val = newyaml(args,keys,label)
            ret = 0
        else:
            (ret, data) = get_data(args)
            if ret:
                print (data, end="")
                return
            if "labels" not in data["metadata"] or label not in data["metadata"]["labels"]:
                print ("Error : label does not match.")
                return
            (ret, val) = get_yaml(args)
        if ret == 0:
            yml = ""
            EDITOR = os.environ.get('EDITOR','vim')
            with tempfile.NamedTemporaryFile(mode='w+t', encoding='utf-8', delete=False, suffix=".tmp") as tf:
                tf.write(val)
                tf.flush()
                subprocess.call([EDITOR, tf.name])
                tf.seek(0)
                yml = tf.read()
                if yml != val:
                    if not keylabelchk(yml,keys,label):
                        print ("Error : label or keys not matched.")
                        return
                    (ret, val) = localCommand("kubectl apply -f " + tf.name + " -n " + args.namespace)
                    print (val, end="")
        else:
            print (val, end="")
    else:
        print ("Error : name not supplied.")

def label_patch(args,label,value):
    if len(args.name) == 0:
        print ("Error : Subject name not given.")
    (ret, val) = localCommand("kubectl patch cm -n " + args.namespace + " -p {\"metadata\":{\"labels\":{\"" + label + "\":\"" + value + "\"}}} " + args.name[0])
    print (val, end="")

# log inputs
log_keys = [
    ["NAME","metadata","name","input-syslog"],
    ["LOGKIND","data","log_kind","syslog"],
    ["PATH","data","path","/var/log/syslog"],
    ["TAG","data","tag","syslog.syslog"]
]
log_label = "logfilter.ssl.com/log"
# procs
proc_keys = [
    ["NAME","metadata","name","proc-os"],
    ["PROC_NAMES","data","proc_names","crond,sshd"],
    ["INTERVAL_SEC","data","interval_sec",'"60"'],
    ["NODE_GROUP","data","node_group","worker"],
    ["TAG","data","tag","proc.*"]
]
proc_label = "logfilter.ssl.com/proc"
# metrics
metric_keys = [
    ["NAME","metadata","name","metrics-pod"],
    ["METRIC_KIND","data","metric_kind","pod-metrics"],
    ["INTERVAL_SEC","data","interval_sec",'"60"'],
    ["TAG","data","tag","metrics.pod"]
]
metric_label = "logfilter.ssl.com/metric"
# log filters
logfilter_keys = [
    ["NAME","metadata","name","logfilter-kubelet-no-such-file"],
    ["LOGKIND","data","log_kind","container_log"],
    ["LOGNAME","data","log_name","kubelet"],
    ["MESSAGE","data","message","no such file or directory"],
    ["ACTION","data","action","ignore"]
]
logfilter_label = "logfilter.ssl.com/filterdata"
# elasticsearch outputs
output_keys = [
    ["NAME","metadata","name","output-es"],
    ["MATCH","data","match","'*'"],
    ["HOST","data","host","elasticsearch.ns.svc"],
    ["PORT","data","port",'"9200"'],
    ["INDEX_PREFIX","data","index_prefix","k8s_cluster"]
]
output_label = "logfilter.ssl.com/es"
# node groups
nodegroup_keys = [
    ["NAME","metadata","name"],
    ["TOLERATIONS","data","tolerations","controlplane,etcd"],
    ["NODE_SELECTOR","data","node_selector","controlplane"]
]
nodegroup_label = "logfilter.ssl.com/nodegroup"
# OS monitoring
osmon_keys = [
    ["NAME","metadata","name"],
    ["CPU_INTERVAL_SEC","data","cpu_interval_sec",'"60"'],
    ["CPU_TAG","data","cpu_tag","os.cpu"],
    ["MEM_INTERVAL_SEC","data","memory_interval_sec",'"60"'],
    ["MEM_TAG","data","memory_tag","os.memory"],
    ["FS_DF_DIR","data","filesystem_df_dir",'/'],
    ["FS_INTERVAL_SEC","data","filesystem_interval_sec",'"300"'],
    ["FS_TAG","data","filesystem_tag","os.filesystem"],
    ["IO_DISKNAME","data","io_diskname","sda"],
    ["IO_INTERVAL_SEC","data","io_interval_sec",'"60"'],
    ["IO_TAG","data","io_tag","os.io"]
]
osmon_label = "logfilter.ssl.com/os"
# apps
app_keys = [
    ["NAME","metadata","name","input-app-statuses"],
    ["APP_KINDS","data","app_kinds","deployments,daemonsets,statefulsets"],
    ["INTERVAL_SEC","data","interval_sec",'"60"'],
    ["TAG","data","tag","apps.*"]
]
app_label = "logfilter.ssl.com/app"

SUBJECTS = []

usage = """flbctl [command] [subject] name

  [command]
    get       : Show subject's ConfigMap referenced by 'name'. Show list without 'name'.
    create    : Create a subject with 'name'.
    delete    : Delete a subject with 'name'.
    edit      : Edit subject's ConfigMap referenced by 'name'.
    disable   : Disable (not delete) a subject with 'name'.
    enable    : Enable a disabled subject with 'name'.

  [subject]
    all       : Show all subjects with command 'get'.
    log       : Log inputs. 3 log kinds of k8s_pod_log, rke_container_log and syslog.
    logfilter : Log filters before sending them to elasticsearch.
    proc      : Process monitorings.
    metric    : Pod/Node metrics.
    output    : Elasticsearch settings.
    nodegroup : DaemonSets for hatoba-monitoring.
    osmon     : OS monitoring.
    app       : K8s apps monitoring.

  Type 'flbctl [edit|create] [subject] help' for subject's ConfigMap data.

"""
epilog = log_help + logfilter_help + proc_help + metric_help + output_help + nodegroup_help + osmon_help + app_help

# arguments
parser = argparse.ArgumentParser(usage=usage,epilog=epilog,formatter_class=argparse.RawDescriptionHelpFormatter)
parser.add_argument("command", choices=["get", "delete", "create", "disable", "enable", "edit"], help="command")
parser.add_argument("subject", choices=["log", "logfilter", "proc", "metric", "output", "nodegroup", "osmon", "app", "all"], help="subject")
parser.add_argument("name", nargs="*")
parser.add_argument("--namespace", "-n", default="fluent-bit", help="namespace")
parser.add_argument("--debug", "-v", action="store_true", help="debug output")
args = parser.parse_args()
if args.debug:
    print(args)

# edit / create
if args.command == "edit" or args.command == "create":
    create = (args.command == "create")

    if args.subject == "log":
        edit(args,log_keys,log_label,create)

    if args.subject == "proc":
        edit(args,proc_keys,proc_label,create)

    if args.subject == "metric":
        edit(args,metric_keys,metric_label,create)

    if args.subject == "logfilter":
        edit(args,logfilter_keys,logfilter_label,create)

    if args.subject == "output":
        edit(args,output_keys,output_label,create)

    if args.subject == "nodegroup":
        edit(args,nodegroup_keys,nodegroup_label,create)

    if args.subject == "osmon":
        edit(args,osmon_keys,osmon_label,create)

    if args.subject == "app":
        edit(args,app_keys,app_label,create)

# get
if args.command == "get":

    all = (args.subject == "all")

    if args.subject == "log" or all:
        if all:
            print (GREEN + "Log inputs" + ENDC)
        get(args,log_keys,log_label)

    if args.subject == "proc" or all:
        if all:
            print (GREEN + "Proccess monitorings" + ENDC)
        get(args,proc_keys,proc_label)

    if args.subject == "metric" or all:
        if all:
            print (GREEN + "Pod/Node metrics" + ENDC)
        get(args,metric_keys,metric_label)

    if args.subject == "logfilter" or all:
        if all:
            print (GREEN + "Log filters" + ENDC)
        get(args,logfilter_keys,logfilter_label)

    if args.subject == "output" or all:
        if all:
            print (GREEN + "Outputs (Elasticsearch)" + ENDC)
        get(args,output_keys,output_label)

    if args.subject == "nodegroup" or all:
        if all:
            print (GREEN + "Node groups" + ENDC)
        get(args,nodegroup_keys,nodegroup_label)

    if args.subject == "osmon" or all:
        if all:
            print (GREEN + "OS monitoring" + ENDC)
        get(args,osmon_keys,osmon_label)

    if args.subject == "app" or all:
        if all:
            print (GREEN + "K8s apps monitoring" + ENDC)
        get(args,app_keys,app_label)

# delete
if args.command == "delete":
    if len(args.name) == 0:
        print ("Error : Subject name not given.")
    (ret, val) = localCommand("kubectl delete cm " + args.name[0] + " -n " + args.namespace)
    print (val, end="")

# enable/disable
if args.command == "enable" or args.command == "disable":

    enabled = "true"
    if args.command == "disable":
        enabled = "disabled"

    if args.subject == "log":
        label_patch(args, log_label, enabled)

    if args.subject == "proc":
        label_patch(args, proc_label, enabled)

    if args.subject == "metric":
        label_patch(args, metric_label, enabled)

    if args.subject == "logfilter":
        label_patch(args, logfilter_label, enabled)

    if args.subject == "output":
        label_patch(args, output_label, enabled)

    if args.subject == "nodegroup":
        label_patch(args, nodegroup_label, enabled)

    if args.subject == "osmon":
        label_patch(args, osmon_label, enabled)

    if args.subject == "app":
        label_patch(args, app_label, enabled)

exit(0)
