.PHONY: build test lint proto migrate docker-up docker-down k8s-apply tf-init tf-plan tf-validate helm-validate docker-build-ci test-integration ci

SERVICES := api-gateway workflow-engine consumer-worker

build:
	@for svc in $(SERVICES); do \
		go build -o bin/$$svc ./cmd/$$svc; \
	done

test:
	go test ./... -count=1 -race -timeout 5m

lint:
	go vet ./...

proto:
	protoc -I api/proto \
		--go_out=api/gen/go --go_opt=paths=source_relative \
		--go-grpc_out=api/gen/go --go-grpc_opt=paths=source_relative \
		api/proto/eventflow/v1/common.proto \
		api/proto/eventflow/v1/topic.proto \
		api/proto/eventflow/v1/event.proto \
		api/proto/eventflow/v1/replay.proto \
		api/proto/eventflow/v1/workflow.proto

test-integration:
	CGO_ENABLED=1 go test -tags=integration ./tests/integration/... -count=1 -timeout 15m

tf-validate:
	terraform fmt -check -recursive terraform/
	cd terraform/environments/dev && terraform init -backend=false -input=false && terraform validate -no-color

helm-validate:
	helm lint ./helm/eventflow
	helm template eventflow ./helm/eventflow > /dev/null

docker-build-ci:
	docker build -f docker/Dockerfile.api-gateway -t eventflow/api-gateway:ci .
	docker build -f docker/Dockerfile.consumer-worker -t eventflow/consumer-worker:ci .
	docker build -f docker/Dockerfile.workflow-engine -t eventflow/workflow-engine:ci .

ci: lint test test-integration docker-build-ci helm-validate tf-validate

helm-install:
	helm install eventflow ./helm/eventflow

demo:
	bash scripts/demo.sh

demo-failure:
	bash scripts/demo-failure.sh

demo-replay:
	bash scripts/demo-replay.sh

demo-generator:
	go run ./cmd/demo-generator --events=1000 --failures=10

migrate:
	psql $(DATABASE_URL) -f migrations/001_initial_schema.sql

docker-up:
	docker compose -f docker/docker-compose.yml up -d --build

docker-down:
	docker compose -f docker/docker-compose.yml down -v

k8s-apply:
	kubectl apply -k deployments/k8s/overlays/dev

tf-init:
	cd terraform/environments/dev && terraform init

tf-plan:
	cd terraform/environments/dev && terraform plan
