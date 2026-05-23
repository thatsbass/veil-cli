.PHONY: build install dev tidy test lint

BINARY_NAME=veil
BIN_DIR=bin

build:
	go build -ldflags="-w -s" -o $(BIN_DIR)/$(BINARY_NAME) cmd/main.go

install: build
	sudo cp $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

dev:
	go run cmd/main.go

tidy:
	go mod tidy

test:
	go test ./... -v -cover

lint:
	golangci-lint run ./...
