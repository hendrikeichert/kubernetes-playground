# Kafka
https://phoenixnap.com/kb/kafka-on-kubernetes

## create namespace
kubectl apply -f namespaces.yaml 
## apply zookeeper
kubectl apply -f zookeeper.yaml 
## apply kafka
kubectl apply -f kafka.yaml
## test kafka with kcat - apply kcat
kubectl apply -f kcat.yaml

Find the kcat pod and copy its name. Enter the pod by executing the following command:
```bash
# kcat-868fd95886-4wpvr
kubectl exec --stdin --tty pods/kcat-868fd95886-4wpvr -- /bin/sh
kubectl exec --stdin --tty [pod-name] -- /bin/sh

# Enter the command below to send Kafka a test message to ingest:
echo "Test Message" | kcat -P -b kafka:29092 -t testtopic -p -1

# Switch to the consumer role and query Kafka for messages by typing:
kcat -C -b kafka:29092 -t testtopic -p -1
```

Test kafka-connect
```bash
kubectl port-forward svc/kafka-connect 8083:8083
curl http://localhost:8083/connectors
```
## confluent, kafka connect
https://packages.confluent.io/


# MongoDB

## build docker image with adminer and support for mongodb
docker build -t adminer-mongodb:latest adminer_mongodb/.
docker buildx build -t adminer-mongodb:latest adminer_mongodb/.
minikube image load adminer-mongodb:latest