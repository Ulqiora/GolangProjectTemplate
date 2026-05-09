set shell := ["bash", "-euo", "pipefail", "-c"]

compose_files := "-f ./deployment/docker-compose.database.yaml -f ./deployment/docker-compose.kafka.yaml -f ./deployment/docker-compose.clickhouse.yaml -f ./deployment/docker-compose.tracing.yaml -f ./deployment/docker-compose.app.yaml"
compose := "docker compose " + compose_files
build_dir := "./builds"
kube_namespace := "auth-local"
kube_app_image := "auth-service:local"

default:
    @just --list

generate-api:
    ./scripts/genproto.sh

test:
    go test ./...

test-pkg:
    go test ./pkg/... -count=1

test-internal:
    go test ./internal/... ./cmd/... -count=1

run-auth config="./configs/auth/local.yaml":
    AUTH_CONFIG_PATH={{config}} go run ./cmd/auth

run-outbox config="./configs/auth/local.yaml":
    AUTH_CONFIG_PATH={{config}} go run ./cmd/outbox

build-auth:
    mkdir -p {{build_dir}}
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o {{build_dir}}/auth ./cmd/auth

build-outbox:
    mkdir -p {{build_dir}}
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o {{build_dir}}/outbox ./cmd/outbox

compose-up:
    {{compose}} up --build -d

compose-up-infra:
    {{compose}} up -d postgres migrate zookeeper kafka schema-registry kafka-ui kafka-exporter kafka-init clickhouse jaeger prometheus grafana node-exporter postgres-exporter

compose-down:
    {{compose}} down -v

compose-logs:
    {{compose}} logs -f auth vk-consumer migrate postgres kafka schema-registry clickhouse jaeger prometheus grafana

compose-config:
    {{compose}} config

goose-create name:
    goose -dir=migrations/postgres create {{name}} sql

minikube-start:
    minikube start --driver=docker

minikube-build:
    minikube image build -t {{kube_app_image}} -f deployment/auth/Dockerfile .

minikube-up: minikube-start minikube-build
    kubectl apply -f deploy/minikube/namespace.yaml
    kubectl apply -f deploy/minikube/postgres.yaml
    kubectl delete job kafka-init -n {{kube_namespace}} --ignore-not-found
    kubectl delete deployment kafka -n {{kube_namespace}} --ignore-not-found
    kubectl apply -f deploy/minikube/kafka.yaml
    kubectl apply -f deploy/minikube/jaeger.yaml
    kubectl rollout status deployment/postgres -n {{kube_namespace}} --timeout=180s
    kubectl rollout status deployment/zookeeper -n {{kube_namespace}} --timeout=180s
    kubectl rollout status deployment/kafka -n {{kube_namespace}} --timeout=240s
    kubectl apply -f deploy/minikube/schema-registry.yaml
    kubectl rollout status deployment/schema-registry -n {{kube_namespace}} --timeout=180s
    kubectl apply -f deploy/minikube/kafka-init.yaml
    kubectl wait --for=condition=complete job/kafka-init -n {{kube_namespace}} --timeout=180s
    kubectl create configmap clickhouse-init --from-file=migrations/clickhouse -n {{kube_namespace}} --dry-run=client -o yaml | kubectl apply -f -
    kubectl apply -f deploy/minikube/clickhouse.yaml
    kubectl rollout status deployment/clickhouse -n {{kube_namespace}} --timeout=240s
    kubectl create configmap migrations --from-file=migrations/postgres -n {{kube_namespace}} --dry-run=client -o yaml | kubectl apply -f -
    kubectl delete job migrate -n {{kube_namespace}} --ignore-not-found
    kubectl apply -f deploy/minikube/migrate.yaml
    kubectl wait --for=condition=complete job/migrate -n {{kube_namespace}} --timeout=180s
    kubectl apply -f deploy/minikube/auth.yaml
    kubectl apply -f deploy/minikube/vk-consumer.yaml
    kubectl rollout status deployment/auth -n {{kube_namespace}} --timeout=240s
    kubectl rollout status deployment/vk-consumer -n {{kube_namespace}} --timeout=240s

minikube-down:
    kubectl delete namespace {{kube_namespace}} --ignore-not-found

minikube-status:
    kubectl get pods,svc,jobs -n {{kube_namespace}} -o wide

minikube-logs app="auth":
    kubectl logs -n {{kube_namespace}} -l app={{app}} --tail=200 -f

minikube-port-forward:
    kubectl port-forward -n {{kube_namespace}} svc/auth 10000:10000 10001:10001 9090:9090 6060:6060

clickhouse-users:
    kubectl exec -n {{kube_namespace}} deploy/clickhouse -- clickhouse-client --query "SELECT user_id, provider, occurred_at FROM users_created ORDER BY occurred_at DESC LIMIT 20"
