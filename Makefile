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
NETWORK_NAME=$(PROJECT_NAME)

.PHONY:all
all: gen-tls-ca


# --------------------SERVICES-UP
dependup:
	if ! docker network inspect ${NETWORK_NAME} >/dev/null 2>&1; then \
        docker network create ${NETWORK_NAME}; \
    fi
	NETWORK_NAME=$(NETWORK_NAME) docker-compose -f $(DC_FOLDER)/docker-compose.database.yaml up --build -d
	NETWORK_NAME=$(NETWORK_NAME) docker-compose -f $(DC_FOLDER)/docker-compose.tracing.yaml up --build -d
	NETWORK_NAME=$(NETWORK_NAME) docker-compose -f $(DC_FOLDER)/docker-compose.kafka.yaml up --build -d
# --------------------GENERATE-GOLANG-GEN-GO
generate-api:
	mkdir -p ${PROTO_TARGET}
	protoc --proto_path $(GRPC_SOURCE) \
    	--go_out=$(PROTO_TARGET) --go_opt=paths=source_relative \
    	--plugin=protoc-gen-go=$(GRPC_UTILS_FOLDER)/protoc-gen-go \
    	--go-grpc_out=$(PROTO_TARGET) --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(GRPC_UTILS_FOLDER)/protoc-gen-go-grpc \
    	--grpc-gateway_out=$(PROTO_TARGET) --grpc-gateway_opt=paths=source_relative \
    	--plugin=protoc-gen-grpc-gateway=$(GRPC_UTILS_FOLDER)/protoc-gen-grpc-gateway \
    	$(GRCP_PROTO_FOLDERS)


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

