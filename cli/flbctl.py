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
    (ret, val) = localCommand("kubectl get cm " + args.name[0] + " -o yaml -n fluent-bit")
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
    (ret, val) = localCommand("kubectl get cm -o json -n fluent-bit " + args.name[0])
    if ret == 0:
        return (0, json.loads(val))
    else:
        return (ret, val)

def get(args,keys,label):
    if len(args.name) > 0:
        (ret, val) = get_yaml(args)
        print (val, end="")
    else:
        (ret, val) = localCommand("kubectl get cm -o json -n fluent-bit")
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
    yml += "  namespace: fluent-bit\ndata:\n"
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

def edit(args,keys,label,create):
    if len(args.name) > 0:
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
                    (ret, val) = localCommand("kubectl apply -f " + tf.name)
                    print (val, end="")
        else:
            print (val, end="")
    else:
        print ("Error : name not supplied.")

def label_patch(args,label,value):
    if len(args.name) == 0:
        print ("Error : Subject name not given.")
    (ret, val) = localCommand("kubectl patch cm -n fluent-bit -p {\"metadata\":{\"labels\":{\"" + label + "\":\"" + value + "\"}}} " + args.name[0])
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
    ["HOST","data","host","elasticsearch.fluent-bit.svc"],
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

SUBJECTS = []

# 引数処理
parser = argparse.ArgumentParser()
parser.add_argument("command", choices=["get", "delete", "create", "disable", "enable", "edit"], help="command")
parser.add_argument("subject", choices=["log", "logfilter", "proc", "metric", "output", "nodegroup", "all"], help="subject")
parser.add_argument("name", nargs="*")
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

# delete
if args.command == "delete":
    if len(args.name) == 0:
        print ("Error : Subject name not given.")
    (ret, val) = localCommand("kubectl delete cm -n fluent-bit " + args.name[0])
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

exit(0)
