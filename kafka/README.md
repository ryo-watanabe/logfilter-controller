## helm install
```
$ helm repo add confluentinc https://confluentinc.github.io/cp-helm-charts/
$ helm repo update
$ helm install confluentinc/cp-helm-charts --name confluent --namespace kafka -f kafka/values.yaml
```
## ssl certs
Generate certs
```
$ cd kafka/certs
$ sh kafka_certs.sh
```
Create secret for kafka broker
```
$ kubectl apply -f kafka-secrets.yaml
```
## Configure kafka-broker and kafka-connect
```
$ kubectl delete -f kafka/kafka.yaml
$ kubectl apply -f kafka/kafka.yaml
$ kubectl apply -f kafka/kafka-connect.yaml
```
### Changes will be applied
#### kafka broker
command:
```
      - command:
        - sh
        - -exc
        - |
          export KAFKA_BROKER_ID=${POD_NAME##*-} && \
          export KAFKA_ZOOKEEPER_CONNECT=${CONFLUENT_CP_ZOOKEEPER_SERVICE_HOST}:2181 && \
          export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://${POD_IP}:9092,SSL://${POD_IP}:9093,EXTERNAL://${HOST_IP}:$((31090 + ${KAFKA_BROKER_ID})) && \
          exec /etc/confluent/docker/run
```
spec:
```
      hostNetwork: true
```
env:
```
        - name: POD_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.name

        - name: KAFKA_SSL_KEYSTORE_FILENAME
          value: kafka.broker.keystore.jks
        - name: KAFKA_SSL_KEYSTORE_CREDENTIALS
          value: keystore-creds
        - name: KAFKA_SSL_TRUSTSTORE_FILENAME
          value: kafka.broker.truststore.jks
        - name: KAFKA_SSL_TRUSTSTORE_CREDENTIALS
          value: truststore-creds
        - name: KAFKA_SSL_KEY_CREDENTIALS
          value: key-creds
        - name: KAFKA_SECURITY_INTER_BROKER_PROTOCOL
          value: PLAINTEXT
        - name: KAFKA_SSL_ENDPOINT_IDENTIFICATION_ALGORITHM
          value: ' '
        - name: KAFKA_SSL_CLIENT_AUTH
          value: required
        - name: KAFKA_AUTHORIZER_CLASS_NAME
          value: kafka.security.auth.SimpleAclAuthorizer
        - name: KAFKA_LISTENER_SECURITY_PROTOCOL_MAP
          value: SSL:SSL,PLAINTEXT:PLAINTEXT,EXTERNAL:SSL
        - name: KAFKA_ALLOW_EVERYONE_IF_NO_ACL_FOUND
          value: "true"
```
volumeMounts:
```
        - mountPath: /etc/kafka/secrets
          name: secrets
```
volumes:
```
      - name: secrets
        secret:
          secretName: kafka-broker-secrets
```
#### kafka connect
command:
```
      - command:
        - sh
        - -exc
        - |
          export CONNECT_BOOTSTRAP_SERVERS=${CONFLUENT_CP_KAFKA_SERVICE_HOST}:9092 && \
          exec /etc/confluent/docker/run

```
env:
```
        - name: CONNECT_INTERNAL_KEY_CONVERTER
          value: org.apache.kafka.connect.json.JsonConverter
        - name: CONNECT_INTERNAL_VALUE_CONVERTER
          value: org.apache.kafka.connect.json.JsonConverter
        - name: CONNECT_KEY_CONVERTER
          value: org.apache.kafka.connect.json.JsonConverter
        - name: CONNECT_KEY_CONVERTER_SCHEMAS_ENABLE
          value: "false"
        - name: CONNECT_VALUE_CONVERTER
          value: org.apache.kafka.connect.json.JsonConverter
        - name: CONNECT_VALUE_CONVERTER_SCHEMAS_ENABLE
          value: "false"

```
## Add elasticsearch connector
```
curl -XPOST -H "Content-type: application/json" http://confluent-cp-kafka-connect.kafka:8083/connectors -d @kafka/es-connect.json
```
es-connect.json (example)
```
{
  "name": "elasticsearch-sink",
  "config": {
    "connector.class": "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
    "tasks.max": "1",
    "flush.timeout.ms": "60000",
    "read.timeout.ms": "10000",
    "topics.regex": "es_.*",
    "transforms": "TimestampRouter",
    "transforms.TimestampRouter.type": "org.apache.kafka.connect.transforms.TimestampRouter",
    "transforms.TimestampRouter.topic.format": "${topic}-${timestamp}",
    "transforms.TimestampRouter.timestamp.format": "yyyy.MM.dd",
    "key.ignore": "true",
    "schema.ignore": "true",
    "connection.url": "http://elasticsearch:9200",
    "type.name": "kafka-connect",
    "name": "elasticsearch-sink"
  }
}
```
Topics which has prefix 'es_' will be transfered to ES as index es_XXXXX-yyyy.MM.dd
## Check kafka broker
```
$ kubectl apply -f kafka/testclient.yaml
$ kubectl exec -it -n kafka testclient -- /bin/bash
```
#### List topics
```
/opt/kafka# bin/kafka-topics.sh --list --zookeeper confluent-cp-zookeeper-headless:2181
```
#### Consume topic stream
```
/opt/kafka# bin/kafka-console-consumer.sh --bootstrap-server confluent-cp-kafka-headless:9092 --topic [topic name]
```
#### Delete topic
```
/opt/kafka# bin/kafka-topics.sh --delete --zookeeper confluent-cp-zookeeper-headless:2181 --topic [topic name]
```
## Client (logfilter-controller) settings
Create secret for kafka client
```
$ kubectl apply -f kafka/certs/client-secret.yaml
```
Controller options
```
logfilter-controller \
 --fluentbitimage=[fluentbit-curl-jq]:[tag] \
 --metricsimage=[fluentbit-curl-jq]:[tag] \
 --kafkasecret=kafka-client-cert \
 --kafkasecretpath=/certs \
 --namespace=fluent-bit
```
Use fluentbit-logfilter >= 0.8 with fluent-bit-curl-jq >= 0.3

Add config map:
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: output-kafka
  namespace: fluent-bit
  labels:
    logfilter.ssl.com/kafka: "true"
data:
  match: "*"
  brokers: 10.10.10.1:9093,10.10.10.2:9093,10.10.10.3:9093
  timestamp_format: iso8601
  topics: es_cluster01
  rdkafka_options: ssl.key.location=/certs/private-key.pem,ssl.certificate.location=/certs/cert-signed.pem,ssl.ca.location=/certs/ca-cert.pem,ssl.key.password=kafka1234,security.protocol=ssl
```
will be set in fluent-bit.conf as below...
```
[OUTPUT]
    Name        kafka
    Match       *
    Timestamp_Format iso8601
    Brokers     10.10.10.1:9093,10.10.10.2:9093,10.10.10.3:9093
    Topics      es_cluster01
    rdkafka.ssl.key.location         /certs/private-key.pem
    rdkafka.ssl.certificate.location /certs/cert-signed.pem
    rdkafka.ssl.ca.location          /certs/ca-cert.pem
    rdkafka.ssl.key.password         kafka1234
    rdkafka.security.protocol        ssl
```
