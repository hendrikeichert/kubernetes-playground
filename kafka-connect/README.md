# Docker build
docker build -t custom-kafka-connect:0.0.3 kafka-connect/docker/

# import image into k3s containerd
docker save custom-kafka-connect:0.0.3 | sudo k3s ctr images import -

# port forward
```
kubectl port-forward svc/kafka-connect 8083:8083 -n kafka-connect
```

# show connectors
```
curl localhost:8083/connectors
```

# create connector
```
curl -d @"kafka-connect/docker/jdbc-source-connector.json" \
  -H "Content-Type: application/json" \
  -X POST http://localhost:8083/connectors
```

# create connector from in-cluster template
```
kubectl -n kafka-connect get configmap kafka-connect-postgres-sink -o jsonpath='{.data.postgres-sink-connector\.json}' \
| curl -H "Content-Type: application/json" -X POST --data-binary @- http://localhost:8083/connectors
```

# create Keycloak Postgres CDC connector (Debezium)
```
kubectl -n kafka-connect get configmap kafka-connect-postgres-sink -o jsonpath='{.data.keycloak-cdc-connector\.json}' \
| curl -H "Content-Type: application/json" -X POST --data-binary @- http://localhost:8083/connectors
```

# inspect connector status
```
curl -s http://localhost:8083/connectors/keycloak-cdc-connector/status
```
