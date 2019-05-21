package fluentbitcfg

import (
)

// config body
const fluentbit_config = `[SERVICE]
    Flush        1
    Daemon       Off
    Log_Level    error
    Parsers_File parsers.conf
@INPUTS@FILTERS@OUTPUTS
`
// filters
const hostname_filter = `
[FILTER]
    Name record_modifier
    Match *
    Record hostname ${HOSTNAME}
`
const metrics_filter = `
[FILTER]
    Name lua
    Match metrics.*
    script /fluent-bit/metrics/fluent-bit-metrics.lua
    call cpu_memory_in_number
`
const ignore_filter = `
[FILTER]
    Name lua
    Match *
    script /fluent-bit/filter/funcs.lua
    call ignore_message
`
// log inputs
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
    script /fluent-bit/filter/funcs.lua
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
// process monitoring
const proc_monitoring = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       sh /fluent-bit/os/chk_proc.sh @PROC_NAME /host
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
// os monitorings
const os_cpu = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       sh /fluent-bit/os/chk_cpu.sh /host
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
const os_memory = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       sh /fluent-bit/os/chk_mem.sh /host
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
const os_io = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       sh /fluent-bit/os/chk_io.sh @NAME /host
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
const os_filesystem = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       sh /fluent-bit/os/chk_filesystem.sh @DIR /host
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
// pod/node metrics
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
// k8s app status
const deployment_status = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       curl -k https://kubernetes.default.svc/apis/apps/v1/deployments?pretty=false -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" | jq '.items[] as $app | (if $app.status.replicas != null then $app.status.replicas else 0 end) as $desired | (if $app.status.readyReplicas != null then $app.status.readyReplicas else 0 end) as $ready | {name:$app.metadata.name, namespace:$app.metadata.namespace, desired:$desired, ready:$ready, notready:($desired - $ready)} |@json' -r
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
const statefulset_status = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       curl -k https://kubernetes.default.svc/apis/apps/v1/statefulsets?pretty=false -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" | jq '.items[] as $app | (if $app.status.replicas != null then $app.status.replicas else 0 end) as $desired | (if $app.status.readyReplicas != null then $app.status.readyReplicas else 0 end) as $ready | {name:$app.metadata.name, namespace:$app.metadata.namespace, desired:$desired, ready:$ready, notready:($desired - $ready)} |@json' -r

    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
const daemonset_status = `
[INPUT]
    Name          exec
    Tag           @TAG
    Command       curl -k https://kubernetes.default.svc/apis/apps/v1/daemonsets?pretty=false -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" | jq '.items[] as $app | (if $app.status.desiredNumberScheduled != null then $app.status.desiredNumberScheduled else 0 end) as $desired | (if $app.status.numberReady != null then $app.status.numberReady else 0 end) as $ready | {name:$app.metadata.name, namespace:$app.metadata.namespace, desired:$desired, ready:$ready, notready:($desired - $ready)} |@json' -r
    Interval_Sec  @INTERVAL
    Interval_NSec 0
    Parser        json
`
// output
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
