.PHONY: proto sqlc test build build-all

BINARY_NAME := wc2026_api
CMD_PATH    := ./cmd/server

proto:
	buf generate

sqlc:
	sqlc generate

test:
	go test ./...

build:
	CGO_ENABLED=0 go build -o bin/$(BINARY_NAME) $(CMD_PATH)

build-all:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o bin/$(BINARY_NAME)-linux-arm64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64  $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
