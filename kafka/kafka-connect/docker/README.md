# Docker build
docker build -t custom-kafka-connect .

# port forward
```
kubectl port-forward pods/kafka-connect-755f68bf6d-dhb8d 8083:8083 -n kafka
```

# show connectors
```
curl localhost:8083/connectors
```

# create connector
```
curl -d @"kafka_connect/jdbc-source-connector.json" \
  -H "Content-Type: application/json" \
  -X POST http://localhost:8083/connectors
```
