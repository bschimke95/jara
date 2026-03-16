.PHONY: build lint test test-integration fmt vet tidy all

all: lint test build

build:
	go build ./cmd/jara

lint:
	golangci-lint run ./...

test:
	go test -race ./...

test-integration:
	go test -race -tags integration ./...

fmt:
	gofumpt -w .

vet:
	go vet ./...

tidy:
	go mod tidy
