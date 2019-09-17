# logfilter-controller

## v0.9

- Fluent-bit config can be edited in ConfigMap 'template'
 - daemonset_fluent-bif.conf ... Config for FBs on each node log gathering and os monitorings.
 - deployment_fluent-bit.conf ... Config for a FB gathering k8s metrics and metadata.
 - Restart FB(s) automatically when the template changed.

#### v0.8 -> v0.9
```
$ kubectl apply artifacts/scripts.yaml
$ kubectl create artifacts/templates.yaml

Edit deploy.yaml :
 logfilter-controller > 0.9
 fluentbit-curl-jq > 0.3

$ kubectl apply -f deploy.yaml
```

## v0.8

- Output kafka

## v0.7

- k8s Apps (deployment/statefulset/daemonset) ready pods monitoring
