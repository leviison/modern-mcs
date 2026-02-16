APP=modern-mcs

.PHONY: run build test test-integration fmt vet tidy

run:
	go run ./cmd/server

build:
	go build -o bin/$(APP) ./cmd/server

test:
	go test ./...

test-integration:
	go test ./internal/integration -v

fmt:
	gofmt -w $(shell find . -name '*.go' -type f)

vet:
	go vet ./...

tidy:
	go mod tidy
