.PHONY: help
.PHONY: build build-server build-client build-all-clients
.PHONY: generate-proto generate-mocks generate-cert
.PHONY: migrate-up migrate-down
.PHONY: test
.PHONY: clean clean-mocks clean-proto clean-bin
.PHONY: run-server

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.buildVersion=$(VERSION) -X main.buildDate=$(DATE)

PROTO_DIR := proto/gophkeeper/v1
GEN_DIR := gen/proto/gophkeeper/v1

build: build-server build-client

build-server:
	go build -o bin/gophkeeper-server ./cmd/server

build-client:
	go build -ldflags "$(LDFLAGS)" -o bin/gophkeeper-client ./cmd/client

# build client for all platforms
build-all-clients:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/client-linux-amd64 ./cmd/client
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/client-darwin-amd64 ./cmd/client
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/client-darwin-arm64 ./cmd/client
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/client-windows-amd64.exe ./cmd/client

generate-proto:
	@mkdir -p $(GEN_DIR)
	protoc \
		--proto_path=proto \
		--go_out=gen/proto --go_opt=paths=source_relative \
		--go-grpc_out=gen/proto --go-grpc_opt=paths=source_relative \
		--go_opt=default_api_level=API_OPAQUE \
		$(PROTO_DIR)/auth.proto $(PROTO_DIR)/data.proto

generate-mocks:
	mockery

# Generate TLS certificates (self-signed for development).
generate-cert:
	@mkdir -p cert
	# Generate CA key and cert.
	openssl genrsa -out cert/ca.key 4096
	openssl req -new -x509 -days 3650 -key cert/ca.key -out cert/ca.crt \
		-subj "/C=RU/ST=Moscow/L=Moscow/O=Gophkeeper/CN=Gophkeeper CA"
	# Generate server key and CSR.
	openssl genrsa -out cert/server.key 4096
	openssl req -new -key cert/server.key -out cert/server.csr \
		-subj "/C=RU/ST=Moscow/L=Moscow/O=Gophkeeper/CN=localhost"
	# Create extensions file for SAN.
	echo "subjectAltName=DNS:localhost,IP:127.0.0.1" > cert/server_ext.cnf
	# Sign server cert with CA.
	openssl x509 -req -days 3650 -in cert/server.csr -CA cert/ca.crt -CAkey cert/ca.key \
		-CAcreateserial -out cert/server.crt -extfile cert/server_ext.cnf
	# Clean up.
	rm -f cert/server.csr cert/server_ext.cnf cert/ca.srl
	@echo "Certificates generated in ./cert/"

migrate-up:
	@if [ -z "$(DSN)" ]; then echo "Usage: make migrate-up DSN=postgres://..."; exit 1; fi
	migrate -path migrations -database "$(DSN)" up

migrate-down:
	@if [ -z "$(DSN)" ]; then echo "Usage: make migrate-down DSN=postgres://..."; exit 1; fi
	migrate -path migrations -database "$(DSN)" down

test: generate-mocks
	go test ./... -v -count=1

clean-mocks:
	@echo "Removing generated mocks..."
	# Находим каталоги 'mocks' и удаляем mock_*.go внутри
	@find . -type d -name mocks -prune -exec sh -c ' \
		for d; do \
			find "$$d" -maxdepth 1 -type f -name "mock_*.go" -print -delete; \
			rmdir "$$d" 2>/dev/null || true; \
		done' sh {} +
	@echo "Done."

clean-proto:
	@find gen/proto -name "*.pb.go" -type f -print -delete

clean-bin:
	rm -f -v bin/*

clean: clean-mocks clean-bin clean-proto

run-server:
	go run ./cmd/server

help:
	@echo "Gophkeeper Makefile"
	@echo "=================="
	@echo ""
	@echo "Build targets:"
	@echo "  build              Build both server and client binaries"
	@echo "  build-server       Build server binary only (output: bin/gophkeeper-server)"
	@echo "  build-client       Build client binary only (output: bin/gophkeeper-client)"
	@echo "  build-all-clients  Build client binaries for multiple platforms (linux, darwin, windows)"
	@echo ""
	@echo "Code generation:"
	@echo "  generate-proto     Generate Go code from Protocol Buffer definitions"
	@echo "  generate-mocks     Generate mocks using mockery"
	@echo "  generate-cert      Generate self-signed TLS certificates for development"
	@echo ""
	@echo "Database migrations:"
	@echo "  migrate-up         Apply all up migrations (usage: make migrate-up DSN=postgres://...)"
	@echo "  migrate-down       Rollback all down migrations (usage: make migrate-down DSN=postgres://...)"
	@echo ""
	@echo "Testing and cleanup:"
	@echo "  test               Run all tests with verbose output"
	@echo "  clean-mocks        Remove generated mock files"
	@echo "  clean-proto        Remove generated protobuf files"
	@echo "  clean-bin          Remove all binaries from bin/ directory"
	@echo "  clean              Perform full cleanup (mocks, binaries, protobuf)"
	@echo ""
	@echo "Development:"
	@echo "  run-server         Run server directly from source"
	@echo "  help               Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION            Current version (default: git describe or 'dev')"
	@echo "  DATE               Build date (auto-generated)"