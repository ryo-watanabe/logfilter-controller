#!/bin/sh
set -e

kubectl config set-cluster cluster --server=https://kubernetes.default.svc
kubectl config set-cluster cluster --certificate-authority=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
kubectl config set-credentials apache --token=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
kubectl config set-context context --cluster=cluster --user=apache --namespace=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)
kubectl config use-context context
cp /root/.kube/config /kubecfg
chown apache /kubecfg

if [ "$UI_USER" != "" ] && [ "$UI_PASSWORD" != "" ] ; then
  htpasswd -cb /.htpasswd $UI_USER $UI_PASSWORD
else
  rm -f /var/www/localhost/htdocs/.htaccess
fi

exec httpd -DFOREGROUND
