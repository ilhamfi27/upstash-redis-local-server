PROJECT_NAME = "upstash-redis-local"
BASE=$(shell pwd)
BUILD_DIR=$(BASE)/bin
VERSION ?= v1.0
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_SHA = $(shell git rev-parse --short HEAD)
LDFLAGS = -ldflags="-X main.Version=${VERSION}"
PACKAGE = $(shell go list -m)

.PHONY: clean
clean:
	@echo "> Cleaning Build targets"
	rm -rf bin

.PHONY: deps
deps:
	@echo "> Installing dependencies"
	@go mod tidy
	@go mod download

.PHONY: build
build: deps
	@echo "> Building upstash-redis-local backend Server Binary"
	go build ${LDFLAGS} -o ${BUILD_DIR}/${PROJECT_NAME}
	@echo "> Binary has been built successfully"


.PHONY: build-cli
build-cli: deps
	@echo "> Building upstash-local CLI"
	go build -o ${BUILD_DIR}/upstash-local ./cmd/upstash-local

.PHONY: test
test:
	go test -v ./...

.PHONY: build-all
build-all: build build-cli

.PHONY: run
run: build
	@echo "> Running ${PROJECT_NAME}"
	${BUILD_DIR}/${PROJECT_NAME}