.PHONY: build install test lint clean

build:
	go build -o bin/rendspec ./cmd/rendspec
	go build -o bin/rendspec-mcp ./cmd/rendspec-mcp

install:
	go install ./cmd/rendspec
	go install ./cmd/rendspec-mcp

test:
	go test ./internal/... -v

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
