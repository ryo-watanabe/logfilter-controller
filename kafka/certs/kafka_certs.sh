#!/bin/sh

DAYS=365
PASSWD=kafka1234
IP1=10.10.10.1
IP2=10.10.10.2
IP3=10.10.10.3

#Create server keystore
keytool -keystore kafka.server.keystore.jks -alias localhost -validity $DAYS -genkey -dname "C=JP, L=Setagaya, O=SpwInc CN=kafka.local" -storepass $PASSWD -keypass $PASSWD -ext SAN=ip:$IP1,ip:$IP2,ip:$IP3

#Create CA and truststore
openssl req -new -x509 -keyout ca-key -subj "/C=JP/L=Setagaya/O=SpwInc/CN=kafka.local" -out ca-cert -days $DAYS -passout pass:$PASSWD
keytool -keystore kafka.server.truststore.jks -alias CARoot -import -file ca-cert -storepass $PASSWD -keypass $PASSWD -noprompt
keytool -keystore kafka.client.truststore.jks -alias CARoot -import -file ca-cert -storepass $PASSWD -keypass $PASSWD -noprompt

#Sign server cert and import into keystore
keytool -keystore kafka.server.keystore.jks -alias localhost -certreq -file cert-file -storepass $PASSWD -keypass $PASSWD -noprompt
openssl x509 -req -CA ca-cert -CAkey ca-key -in cert-file -out cert-signed -days $DAYS -CAcreateserial -passin pass:$PASSWD
keytool -keystore kafka.server.keystore.jks -alias CARoot -import -file ca-cert -storepass $PASSWD -keypass $PASSWD -noprompt
keytool -keystore kafka.server.keystore.jks -alias localhost -import -file cert-signed -storepass $PASSWD -keypass $PASSWD -noprompt

#Export private key
keytool -importkeystore -srckeystore kafka.server.keystore.jks -destkeystore kafka.server.keystore.p12 -deststoretype PKCS12 -srcalias localhost -deststorepass $PASSWD -destkeypass $PASSWD -srcstorepass $PASSWD -srckeypass $PASSWD -noprompt
openssl pkcs12 -in kafka.server.keystore.p12 -nocerts -out cert-key -passin pass:$PASSWD -passout pass:$PASSWD

#Kafka broker secrets (ConfigMap)
cp kafka-secrets.yaml.template kafka-secrets.yaml
sed -i "s|TRUSTSTORE_BASE64|$(cat kafka.server.truststore.jks | base64 -w 0)|" kafka-secrets.yaml
sed -i "s|KEYSTORE_BASE64|$(cat kafka.server.keystore.jks | base64 -w 0)|" kafka-secrets.yaml
sed -i "s|PASSWD_BASE64|$(echo -n $PASSWD | base64 -w 0)|g" kafka-secrets.yaml

#Client secrets (ConfigMap)
cp client-secrets.yaml.template client-secrets.yaml
sed -i "s|CA_CERT_BASE64|$(cat ca-cert | base64 -w 0)|" client-secrets.yaml
sed -i "s|CERT_SIGNED_BASE64|$(cat cert-signed | base64 -w 0)|" client-secrets.yaml
sed -i "s|PRIVATE_KEY_BASE64|$(cat cert-key | base64 -w 0)|" client-secrets.yaml
