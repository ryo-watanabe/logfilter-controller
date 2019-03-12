package fluentbitcfg

import (
  "strings"

  corev1 "k8s.io/api/core/v1"
)

const fluentbit_config = `[SERVICE]
    Flush        1
    Daemon       Off
    Log_Level    error
    Parsers_File parsers.conf
@INPUTS@FILTERS@OUTPUTS
`
const hostname_filter = `
[FILTER]
    Name record_modifier
    Match *
    Record hostname ${HOSTNAME}
`
const ignore_filter = `
[FILTER]
    Name lua
    Match *
    script /fluent-bit/etc/funcs.lua
    call ignore_message
`
const k8s_pod_log = `
[INPUT]
    Name             tail
    Path             @PATH
    DB               /var/log/containers/fluent-bit.kube.db
    Parser           docker
    Tag              @TAG
    Refresh_Interval 5
    Mem_Buf_Limit    5MB
    Skip_Long_Lines  On
[FILTER]
    Name             kubernetes
    Match            @TAG
    Kube_URL         https://kubernetes.default.svc:443
    Kube_CA_File     /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    Kube_Token_File  /var/run/secrets/kubernetes.io/serviceaccount/token
`
const rke_container_log = `
[INPUT]
    Name             tail
    Path             @PATH
    DB               /var/lib/rancher/rke/log/fluent-bit.rke.db
    Parser           docker
    Tag              @TAG
    Refresh_Interval 5
    Mem_Buf_Limit    5MB
    Skip_Long_Lines  On
[FILTER]
    Name   lua
    Match  @TAG
    script /fluent-bit/etc/funcs.lua
    call   add_record
`
const syslog = `
[INPUT]
    Name             tail
    Path             @PATH
    DB               /var/log/fluent-bit.syslog.db
    Path_Key         log_name
    Tag              @TAG
    Refresh_Interval 5
    Mem_Buf_Limit    5MB
    Skip_Long_Lines  On
`
const proc_monitoring = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       sh /fluent-bit/etc/chk_proc.sh @PROC_NAME @INTERVAL /host /cache
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
const pod_metrics = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       curl -k https://kubernetes.default.svc/apis/metrics.k8s.io/v1beta1/pods?pretty=false -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" | jq '.items[] as $pod | $pod.containers[] as $container | {name:$pod.metadata.name, container:$container.name, cpu:$container.usage.cpu, memory:$container.usage.memory, namespace:$pod.metadata.namespace} |@json' -r
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
const node_metrics = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       curl -k https://kubernetes.default.svc/apis/metrics.k8s.io/v1beta1/nodes?pretty=false -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" | jq '.items[] | {name:.metadata.name, cpu:.usage.cpu, memory:.usage.memory} |@json' -r
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
const es_output = `
[OUTPUT]
    Name            es
    Match           @MATCH
    Host            @HOST
    Port            @PORT
    Logstash_Format On
    Retry_Limit     False
    Type            flb_type
    Include_Tag_key On
    Logstash_Prefix @PREFIX
`

// Make fluent-bit.conf for DaemonSet
func MakeFluentbitConfig(logs *corev1.ConfigMapList, procs *corev1.ConfigMapList,
  outputs *corev1.ConfigMapList, node_group string) map[string]string {

    ins := ""
    // Log inputs
    for _, log := range logs.Items {
      input := ""
      if log.Data["log_kind"] == "k8s_pod_log" {
        input = k8s_pod_log
      } else if log.Data["log_kind"] == "rke_container_log" {
        input = rke_container_log
      } else if log.Data["log_kind"] == "syslog" {
        input = syslog
      } else {
        continue
      }
      input = strings.Replace(input, "@PATH", log.Data["path"], 1)
      input = strings.Replace(input, "@TAG", log.Data["tag"], 2)
      ins += input
    }
    // Proccess monitorings
    for _, proc := range procs.Items {
      if proc.Data["node_group"] != node_group {
        continue
      }
      proc_names := strings.Split(proc.Data["proc_names"],",")
      for _, proc_name := range proc_names {
        tag := strings.Replace(proc.Data["tag"], "*", proc_name, 1)
        input := proc_monitoring
        input = strings.Replace(input, "@TAG", tag, 1)
        input = strings.Replace(input, "@PROC_NAME", proc_name, 1)
        input = strings.Replace(input, "@INTERVAL", proc.Data["interval_sec"], 2)
        ins += input
      }
    }
    // Outputs
    outs := ""
    for _, out := range outputs.Items {
      output := ""
      output = strings.Replace(output, "@MATCH", out.Data["match"], 1)
      output = strings.Replace(output, "@HOST", out.Data["host"], 1)
      output = strings.Replace(output, "@PORT", out.Data["port"], 1)
      output = strings.Replace(output, "@PREFIX", out.Data["index_prefix"], 1)
      outs += output
    }

    config := fluentbit_config
    config = strings.Replace(config, "@INPUTS", ins, 1)
    config = strings.Replace(config, "@FILTERS", hostname_filter + ignore_filter, 1)
    config = strings.Replace(config, "@OUTPUTS", outs, 1)

    return map[string]string{"fluent-bit.conf":config}
}

// Make fluent-bit.conf for DaemonSet
func MakeFluentbitMetricsConfig(metrics *corev1.ConfigMapList,
  outputs *corev1.ConfigMapList) map[string]string {

    ins := ""
    // K8s metrics inputs
    for _, m := range metrics.Items {
      input := ""
      if m.Data["metric_kind"] == "pod" {
        input = pod_metrics
      } else if m.Data["metric_kind"] == "node" {
        input = node_metrics
      } else {
        continue
      }
      input = strings.Replace(input, "@INTERVAL", m.Data["interval_sec"], 2)
      input = strings.Replace(input, "@TAG", m.Data["tag"], 2)
      ins += input
    }
    // Outputs
    outs := ""
    for _, out := range outputs.Items {
      output := ""
      output = strings.Replace(output, "@MATCH", out.Data["match"], 1)
      output = strings.Replace(output, "@HOST", out.Data["host"], 1)
      output = strings.Replace(output, "@PORT", out.Data["port"], 1)
      output = strings.Replace(output, "@PREFIX", out.Data["index_prefix"], 1)
      outs += output
    }

    config := fluentbit_config
    config = strings.Replace(config, "@INPUTS", ins, 1)
    config = strings.Replace(config, "@FILTERS", hostname_filter, 1)
    config = strings.Replace(config, "@OUTPUTS", outs, 1)

    return map[string]string{"fluent-bit.conf":config}
}
