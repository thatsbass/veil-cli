.PHONY: build install dev tidy

BINARY_NAME=veil
BIN_DIR=bin

build:
	go build -ldflags="-w -s" -o $(BIN_DIR)/$(BINARY_NAME) cmd/main.go

install: build
	cp $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

dev:
	go run cmd/main.go

tidy:
	go mod tidy
