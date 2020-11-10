SHELL=/bin/zsh

VERSION ?= $(shell git describe --tags --always --abbrev=10)
NOW ?= $(shell date)

tidy:
	go mod tidy -v

format:
	gofmt -s -w **/*.go

golint:
	golint -set_exit_status ./...

golangci:
	golangci-lint run -c .golangci.toml

lint: golint golangci


test:
# 	go test -v ./...
	gotestsum -f pkgname-and-test-fails -- ./...

build:
	go build -ldflags "-X \"main.version=$(VERSION)\" -X \"main.buildDate=$(NOW)\"" -v ./cmd/favicon

run:
	go run ./cmd/favicon/main.go

all: tidy format lint test build
checks: tidy format lint
