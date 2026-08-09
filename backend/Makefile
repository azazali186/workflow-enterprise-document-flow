# Go >= 1.24 no longer links against old bytedance/sonic (stdlib linkname
# restrictions); go.mod pins sonic >= v1.15.2 which is compatible. If you run a
# 32-bit host Go toolchain, set GOARCH=amd64 CGO_ENABLED=0 (all deps are
# pure-Go):   make build GOARCH=amd64 CGO_ENABLED=0
export GOARCH ?= amd64
export CGO_ENABLED ?= 0

.PHONY: run build test test-integration vet lint migrate-up swagger tidy docker-up

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

test:
	go test ./... -v -cover

test-integration:
	go test ./... -tags=integration -v

vet:
	go vet ./...

lint:
	golangci-lint run

migrate-up:
	go run cmd/migrate/main.go up

swagger:
	swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

tidy:
	go mod tidy

docker-up:
	docker compose up -d postgres redis nats minio
