all: build
.PHONY: all

SOURCE_GIT_COMMIT ?=$(shell git rev-parse --short "HEAD^{commit}" 2>/dev/null)
SOURCE_GIT_TAG ?=$(shell git describe --long --tags --abbrev=7 --match 'v[0-9]*' 2>/dev/null || echo 'v0.0.1-$(SOURCE_GIT_COMMIT)')
BUILD_TIME = $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
BINARY_NAME = k-mcp
PACKAGE = github.com/ardaguclu/k-mcp
LD_FLAGS = -s -w \
	-X '$(PACKAGE)/pkg/version.version=$(SOURCE_GIT_TAG)' \
	-X '$(PACKAGE)/pkg/version.gitCommit=$(SOURCE_GIT_COMMIT)' \
	-X '$(PACKAGE)/pkg/version.buildDate=$(BUILD_TIME)'

build:
	go build -ldflags "$(LD_FLAGS)" -o $(BINARY_NAME) ./cmd/main.go
