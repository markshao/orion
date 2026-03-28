.PHONY: all build local-install install uninstall run clean github-release

APP_NAME := orion
BUILD_DIR := bin
BIN_PATH := $(BUILD_DIR)/$(APP_NAME)
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X orion/internal/version.Version=$(VERSION) \
	-X orion/internal/version.Commit=$(COMMIT) \
	-X orion/internal/version.Date=$(DATE)
PREFIX ?= /usr/local
INSTALL_DIR ?= $(PREFIX)/bin
INSTALL_PATH := $(INSTALL_DIR)/$(APP_NAME)
GOCACHE ?= $(CURDIR)/.cache/go-build
GOENV := GOCACHE=$(GOCACHE)

all: build

build:
	@mkdir -p $(BUILD_DIR)
	@mkdir -p $(GOCACHE)
	$(GOENV) go build -ldflags "$(LDFLAGS)" -o $(BIN_PATH) main.go

local-install: build
	sudo install -m 0755 $(BIN_PATH) $(INSTALL_PATH)

install: local-install

uninstall:
	sudo rm -f $(INSTALL_PATH)

test:
	@mkdir -p $(GOCACHE)
	$(GOENV) go test ./...

run: build
	./$(BIN_PATH)

clean:
	rm -rf $(BUILD_DIR)

github-release:
	@if [ -z "$(tag)" ]; then \
		echo "Usage: make github-release tag=v1.0.0-alpha.11"; \
		exit 1; \
	fi
	@branch=$${branch:-main}; \
	orion sync-ref --branch $$branch; \
	echo "Tagging refs/heads/$$branch as $(tag)"; \
	orion run git tag $(tag) refs/heads/$$branch; \
	echo "Pushing tag $(tag)"; \
	orion run git push origin $(tag)
