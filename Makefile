BINARY := hack
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
GO := go

.PHONY: build install clean test lint fmt server

build:
	$(GO) build $(LDFLAGS) -o bin/$(BINARY) ./cmd/hack

install:
	$(GO) install $(LDFLAGS) ./cmd/hack

server:
	$(GO) build $(LDFLAGS) -o bin/hack-server ./server/cmd/server

clean:
	rm -rf bin/

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...
	goimports -w .

run:
	$(GO) run $(LDFLAGS) ./cmd/hack

run-server:
	$(GO) run ./server/cmd/server

.DEFAULT_GOAL := build
