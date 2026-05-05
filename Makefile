# Makefile for external-dns-webhook-technitium
# Note: We recommend using 'just' for task automation, but this Makefile is provided for compatibility

.PHONY: help build test docker-build init clean

help:
	@echo "Available targets:"
	@echo "  build         - Build the webhook binary"
	@echo "  test          - Run tests"
	@echo "  docker-build  - Build Docker image"
	@echo "  init          - Initialize Go modules"
	@echo "  clean         - Clean build artifacts"
	@echo ""
	@echo "Note: For a better experience, use 'just' instead of make"
	@echo "Run 'just --list' to see all available commands"

build:
	go build -o bin/webhook ./cmd/webhook

test:
	go test -v ./...

docker-build:
	docker build -t external-dns-webhook-technitium:latest .

init:
	go mod tidy

clean:
	rm -rf bin/
	go clean
