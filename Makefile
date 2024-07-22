SECRET_KEYS_FOLDER=./.secret_keys
all: gen-tls-ca

.PHONY:gen-tls-ca
gen-mtls-ca:
	openssl genrsa -out ca.key 2048
	openssl req -new -x509 -key ca.key -out ca.crt -subj "/CN=my-ca"

.PHONY:gen-mtls-certificate-server
gen-mtls-certificate-server:
	openssl genrsa -out server.key 2048
	openssl req -new -key server.key -out server.csr -subj "/CN=my-server"
	openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

.PHONY:gen-mtls-certificate-client
gen-mtls-certificate-client:
	openssl genrsa -out client.key 2048
	openssl req -new -key client.key -out client.csr -subj "/CN=my-client"
	openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt

.PHONY:gen-tls-ca
gen-tls-ca:
	openssl genrsa -out $(SECRET_KEYS_FOLDER)/server.key 2048
	openssl req -new -key $(SECRET_KEYS_FOLDER)/server.key -out $(SECRET_KEYS_FOLDER)/server.csr -subj "/CN=localhost"
	openssl x509 -req -days 365 -in $(SECRET_KEYS_FOLDER)/server.csr -signkey $(SECRET_KEYS_FOLDER)/server.key -out $(SECRET_KEYS_FOLDER)/server.crt
