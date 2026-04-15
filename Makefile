# secret keys
SECRET_KEYS_FOLDER=./.secret_keys
# build path (FOR GRPC UTILITIES)
GO_BIN := $(shell go env GOPATH)/bin
GRPC_UTILS_FOLDER := $(GO_BIN)
PROTO_TARGET=./internal/api-grpc
GRPC_SOURCE=api
GRCP_PROTO_FOLDERS=$(shell find $(GRPC_SOURCE) -type f)
# DEPLOYMENTS
DC_FOLDER=./deployment
PROJECT_NAME=MYSERVICE
BUILD_EXEC?=outbox
NETWORK_NAME=$(PROJECT_NAME)

.PHONY:all
all: dependup


# --------------------SERVICES-UP
.PHONY: dependup
dependup:
	if ! docker network inspect ${NETWORK_NAME} >/dev/null 2>&1; then \
        docker network create ${NETWORK_NAME}; \
    fi
	NETWORK_NAME=$(NETWORK_NAME) docker-compose -f $(DC_FOLDER)/docker-compose.database.yaml up --build -d
	NETWORK_NAME=$(NETWORK_NAME) docker-compose -f $(DC_FOLDER)/docker-compose.kafka.yaml up --build -d
	NETWORK_NAME=$(NETWORK_NAME) docker-compose -f $(DC_FOLDER)/docker-compose.tracing.yaml up --build -d
	#sleep 5
	#NETWORK_NAME=$(NETWORK_NAME) DOCKER_IMAGE=${BUILD_EXEC} docker-compose -f $(DC_FOLDER)/docker-compose.app.yaml up --build -d
# --------------------GENERATE-GOLANG-GEN-GO
.PHONY: .load-proto-bins
.load-proto-bins:
	./scripts/getprotobins.sh

.PHONY: generate-api
generate-api: .load-proto-bins .deps
	./scripts/genproto.sh


# --------------------GENERATE-TLS-CERTIFICATE
.PHONY:gen-tls-ca
gen-mtls-ca:
	@mkdir -p $(SECRET_KEYS_FOLDER)
	@openssl genrsa -out ca.key 2048
	@openssl req -new -x509 -key ca.key -out ca.crt -subj "/CN=my-ca"
	@make gen-mtls-certificate-server & make gen-mtls-certificate-client

.PHONY:gen-mtls-certificate-server
gen-mtls-certificate-server:
	@mkdir -p $(SECRET_KEYS_FOLDER)
	@openssl genrsa -out server.key 2048
	@openssl req -new -key server.key -out server.csr -subj "/CN=my-server"
	@openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

.PHONY:gen-mtls-certificate-client
gen-mtls-certificate-client:
	@mkdir -p $(SECRET_KEYS_FOLDER)
	@openssl genrsa -out client.key 2048
	@openssl req -new -key client.key -out client.csr -subj "/CN=my-client"
	@openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt

.PHONY:gen-tls-ca
gen-tls-ca:
	@mkdir -p $(SECRET_KEYS_FOLDER)
	@openssl genrsa -out $(SECRET_KEYS_FOLDER)/server.key 2048
	@openssl req -new -key $(SECRET_KEYS_FOLDER)/server.key -out $(SECRET_KEYS_FOLDER)/server.csr -subj "/CN=localhost"
	@openssl x509 -req -days 365 -in $(SECRET_KEYS_FOLDER)/server.csr -signkey $(SECRET_KEYS_FOLDER)/server.key -out $(SECRET_KEYS_FOLDER)/server.crt

# --------------------BUILD-APPLICATION
.PHONY:docker-image
docker-image:
	docker build -f ./deployment/${BUILD_EXEC}/Dockerfile --rm -t ${BUILD_EXEC} . --build-arg PROJECT=${BUILD_EXEC}

.PHONY:build
build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o builds/${BUILD_EXEC}/${BUILD_EXEC} ./cmd/${BUILD_EXEC}
.PHONY:debug
debug: build
	dlv debug ./cmd/${BUILD_EXEC} --headless --listen=:2345 --api-version=2

# -------------------- External dependencies
.PHONY: goose-create
goose-create:
	goose -dir=migrations create ${migration-name} sql

.PHONY: .deps
.deps: .deps-googleapis .deps-protobuf .deps-validate

.PHONY: .deps-googleapis
.deps-googleapis:
	@echo "Downloading Google APIs..."
	@mkdir -p third_party/googleapis
	@git clone --depth 1 https://github.com/googleapis/googleapis.git third_party/googleapis-tmp || true
	@cp -r third_party/googleapis-tmp/google third_party/ || true
	@rm -rf third_party/googleapis-tmp

.PHONY: .deps-protobuf
.deps-protobuf:
	@echo "Downloading Protobuf well-known types..."
	@mkdir -p third_party/protobuf
	@git clone --depth 1 https://github.com/protocolbuffers/protobuf.git third_party/protobuf-tmp || true
	@cp -r third_party/protobuf-tmp/src/google/protobuf third_party/ || true
	@rm -rf third_party/protobuf-tmp

.PHONY: .deps-validate
.deps-validate:
	@echo "Downloading Validate annotations..."
	@mkdir -p third_party/validate
	@git clone --depth 1 https://github.com/bufbuild/protoc-gen-validate.git third_party/validate-tmp || true
	@cp third_party/validate-tmp/validate/validate.proto third_party/validate/ || true
	@rm -rf third_party/validate-tmp