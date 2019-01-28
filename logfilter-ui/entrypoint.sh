#!/bin/sh

kubectl config set-cluster cluster --server=https://kubernetes.default.svc
kubectl config set-cluster cluster --certificate-authority=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
kubectl config set-credentials apache --token=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
kubectl config set-context context --cluster=cluster --user=apache
kubectl config use-context context
cp /root/.kube/config /kubecfg
chown apache /kubecfg

exec httpd -DFOREGROUND
