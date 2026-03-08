# Kubernetes Playground

This repository is organized as one folder per service and deployed via root Kustomize.

## Deploy

Apply everything currently enabled in root kustomization:

```bash
kubectl apply -k .
kubectl get pods -A
```

## Services currently enabled

The root `kustomization.yaml` currently includes:

- `adminer`
- `confluent` (Zookeeper, Kafka, Schema Registry)
- `elasticsearch`
- `grafana`
- `kafka`
- `kafka-connect`
- `kcat`
- `keycloak`
- `mongodb`
- `otel`
- `postgresql`
- `prometheus`
- `redpanda`

## Services present but not enabled by root apply

- `camunda`
- `dashboard-api`
- `dex`
- `spicedb` (currently commented in root kustomization)
- `webui`

You can still deploy those individually, for example:

```bash
kubectl apply -k spicedb
kubectl apply -k webui
```

## Core internal endpoints

- PostgreSQL: `postgresql.postgresql.svc.cluster.local:5432`
- Confluent Kafka: `kafka.confluent.svc.cluster.local:9092`
- Confluent Schema Registry: `schema-registry.confluent.svc.cluster.local:8081`
- Kafka Connect: `kafka-connect.kafka-connect.svc.cluster.local:8083`
- Redpanda Console: `redpanda-console.redpanda.svc.cluster.local:8080`
- Elasticsearch: `elasticsearch.elasticsearch.svc.cluster.local:9200`
- Keycloak: `keycloak.keycloak.svc.cluster.local:8080`

## Streaming smoke test

Enter the `kcat` pod:

```bash
kubectl get pods -n kcat
kubectl exec -n kcat -it <kcat-pod-name> -- /bin/sh
```

Produce and consume against Confluent Kafka:

```bash
echo "Test Message" | kcat -P -b kafka.confluent.svc.cluster.local:9092 -t testtopic -p -1
kcat -C -b kafka.confluent.svc.cluster.local:9092 -t testtopic -p -1 -e
```

## Kafka Connect notes

`kafka-connect` uses a custom image tag `custom-kafka-connect:0.0.3`.

Build and import into k3s containerd:

```bash
docker build -t custom-kafka-connect:0.0.3 kafka-connect/docker/
docker save custom-kafka-connect:0.0.3 | sudo k3s ctr images import -
kubectl rollout restart deployment/kafka-connect -n kafka-connect
```

Connector API quick check:

```bash
kubectl port-forward -n kafka-connect svc/kafka-connect 8083:8083
curl -s http://localhost:8083/connectors
```

## Redpanda Console

Configured to use Confluent Kafka + Schema Registry + Kafka Connect.

```bash
kubectl port-forward -n redpanda svc/redpanda-console 8080:8080
```
