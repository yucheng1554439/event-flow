# EventFlow Deployment Guide

## Helm (Recommended for Kubernetes)

```bash
# Install EventFlow with defaults
helm install eventflow ./helm/eventflow

# Custom values
helm install eventflow ./helm/eventflow \
  --set imageTag=0.2.0 \
  --set replicaCount.apiGateway=3 \
  --set replicaCount.consumerWorker=12 \
  --set kafkaBootstrapServers=kafka.eventflow.svc:9092

# Upgrade
helm upgrade eventflow ./helm/eventflow -f my-values.yaml
```

### Key Values

| Value | Default | Description |
|-------|---------|-------------|
| `replicaCount.apiGateway` | 3 | REST/gRPC gateway replicas |
| `replicaCount.consumerWorker` | 6 | Kafka consumer pool |
| `replicaCount.workflowEngine` | 2 | Workflow orchestrator |
| `imageTag` | latest | Container image tag |
| `kafkaBootstrapServers` | kafka:9092 | Kafka broker list |
| `postgres.enabled` | true | Embed PostgreSQL StatefulSet |
| `redis.enabled` | true | Embed Redis Deployment |
| `resourceLimits.*` | see values.yaml | CPU/memory limits |

### Health Checks

All services expose `/healthz`. API Gateway also exposes `/metrics` for Prometheus.

## Docker Compose (Local)

```bash
make docker-up
```

## Terraform (AWS)

```bash
cd terraform/environments/dev
terraform init
terraform plan -var="db_password=$DB_PASSWORD"
terraform apply -var="db_password=$DB_PASSWORD"
```

Provisions: VPC, EKS, MSK (Kafka), RDS PostgreSQL, ElastiCache Redis.

## CI/CD

GitHub Actions pipeline (`.github/workflows/ci.yml`):

1. `go vet` lint
2. Proto codegen verification
3. Unit tests (`./internal/...`, `./pkg/...`)
4. Integration tests (Testcontainers + Kafka/Postgres/Redis)
5. Docker image builds
6. `helm lint` + `helm template`
7. `terraform fmt -check` + `terraform validate`

## Topic Provisioning in Production

Topics are created via API (REST or gRPC `TopicService.CreateTopic`). The service:

1. Validates configuration
2. Creates topic in Kafka via AdminClient
3. Waits for broker acknowledgment
4. Persists metadata in PostgreSQL

Delete reverses: Kafka `DeleteTopics` then PostgreSQL row removal.
