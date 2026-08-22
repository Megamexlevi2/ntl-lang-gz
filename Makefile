SHELL := /usr/bin/env bash

VERSION := $(shell grep -o '"version"[[:space:]]*:[[:space:]]*"[^"]*"' internal/app/version.json | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

GO_MIN := 1.23
GO_VER := 1.23.6

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
	ifneq ($(wildcard /proc/version),)
		IS_ANDROID := $(shell grep -qi android /proc/version 2>/dev/null && echo yes || echo no)
	else
		IS_ANDROID := no
	endif
	ifeq ($(IS_ANDROID),yes)
		HOST_OS := android
	else
		HOST_OS := linux
	endif
else ifeq ($(UNAME_S),Darwin)
	HOST_OS := macos
else ifneq (,$(findstring MINGW,$(UNAME_S))$(findstring MSYS,$(UNAME_S))$(findstring CYGWIN,$(UNAME_S)))
	HOST_OS := windows
else ifeq ($(UNAME_S),FreeBSD)
	HOST_OS := freebsd
else
	HOST_OS := unknown
endif

ifeq ($(UNAME_M),x86_64)
	HOST_ARCH := amd64
else ifeq ($(UNAME_M),amd64)
	HOST_ARCH := amd64
else ifeq ($(UNAME_M),aarch64)
	HOST_ARCH := arm64
else ifeq ($(UNAME_M),arm64)
	HOST_ARCH := arm64
else ifeq ($(UNAME_M),riscv64)
	HOST_ARCH := riscv64
else
	HOST_ARCH := unknown
endif

OS ?= $(HOST_OS)
ARCH ?= $(HOST_ARCH)

ifeq ($(OS),macos)
	GOOS := darwin
else ifeq ($(OS),android)
	GOOS := linux
else
	GOOS := $(OS)
endif

ifeq ($(ARCH),x86_64)
	GOARCH := amd64
else ifeq ($(ARCH),aarch64)
	GOARCH := arm64
else
	GOARCH := $(ARCH)
endif

ifeq ($(OS),windows)
	EXT := .exe
else
	EXT :=
endif

BIN := lunex$(EXT)
RELEASE_DIR := release

GOBIN := $(shell command -v go 2>/dev/null)

RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m
BOLD := \033[1m

.PHONY: all build release test clean install-go banner help

all: build

banner:
	@printf '\n  Lunex lang  v$(VERSION)\n  Created by David Dev\n  GitHub: https://github.com/Megamexlevi2\n\n'

install-go:
	@if [ -n "$(GOBIN)" ] && printf '%s\n%s\n' "$(GO_MIN)" "$$(go version | awk '{print $$3}' | sed 's/go//')" | sort -V | head -n1 | grep -qx "$(GO_MIN)"; then \
		printf '$(GREEN)[ok]$(NC) Go '"$$(go version | awk '{print $$3}')"' is already installed\n'; \
	else \
		printf '$(BLUE)==>$(NC) $(BOLD)Installing Go $(GO_MIN)+$(NC)\n'; \
		tmp=$$(mktemp -d); \
		curl -fsSL "https://go.dev/dl/go$(GO_VER).$(GOOS)-$(GOARCH).tar.gz" -o "$$tmp/go.tar.gz"; \
		tar -C "$$tmp" -xzf "$$tmp/go.tar.gz"; \
		mkdir -p "$(HOME)/.local/bin"; \
		cp -f "$$tmp/go/bin/go" "$(HOME)/.local/bin/go"; \
		printf '$(GREEN)[ok]$(NC) Go $(GO_VER) installed\n'; \
	fi

build: banner install-go
	@printf '$(BLUE)==>$(NC) $(BOLD)Building Lunex for $(GOOS)/$(GOARCH)$(NC)\n'
	@GONOSUMDB='*' GOFLAGS='-mod=mod' GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 \
		go build -trimpath -tags netgo -ldflags="-s -w" -o "$(BIN)" ./cmd/lunex
	@test -f "$(BIN)" || (printf '$(RED)[error]$(NC) Build completed without producing $(BIN)\n' >&2; exit 1)
	@printf '$(GREEN)[ok]$(NC) $(BIN) built: '"$$(du -sh "$(BIN)" | cut -f1)"'\n'
	@printf '\n$(GREEN)[ok]$(NC) Lunex v$(VERSION) built successfully\n'
	@printf '$(CYAN)[info]$(NC) Run:  ./$(BIN) run <file.lx>\n'

release: banner install-go
	@printf '$(BLUE)==>$(NC) $(BOLD)Starting multi-platform release build$(NC)\n'
	@mkdir -p $(RELEASE_DIR)
	@for target in linux:amd64 linux:arm64 windows:amd64 darwin:amd64 darwin:arm64 android:arm64 freebsd:amd64; do \
		t_os=$${target%%:*}; t_arch=$${target##*:}; \
		ext=""; \
		[ "$$t_os" = "windows" ] && ext=".exe"; \
		name="lunex-$$t_os-$$t_arch$$ext"; \
		printf '$(CYAN)[info]$(NC) Building '"$$t_os/$$t_arch"'\n'; \
		GONOSUMDB='*' GOFLAGS='-mod=mod' GOOS=$$t_os GOARCH=$$t_arch CGO_ENABLED=0 \
			go build -trimpath -tags netgo -ldflags="-s -w" -o "$(RELEASE_DIR)/$$name" ./cmd/lunex || exit 1; \
		printf '$(GREEN)[ok]$(NC) Done: $(RELEASE_DIR)/'"$$name ($$(du -sh "$(RELEASE_DIR)/$$name" | cut -f1))"'\n'; \
	done
	@printf '$(GREEN)[ok]$(NC) All targets built successfully in ./$(RELEASE_DIR)/\n'

test: build
	@if [ -f tests/run_tests.sh ]; then \
		printf '$(BLUE)==>$(NC) $(BOLD)Running Lunex integration tests$(NC)\n'; \
		bash tests/run_tests.sh; \
	fi
	@printf '$(GREEN)[ok]$(NC) All tests passed\n'

clean:
	@printf '$(BLUE)==>$(NC) $(BOLD)Cleaning build artifacts$(NC)\n'
	@rm -rf lunex lunex.exe $(RELEASE_DIR)
	@printf '$(GREEN)[ok]$(NC) Cleaned\n'

help:
	@echo "Targets:"
	@echo "  build    - build for current platform (default)"
	@echo "  release  - build for all platforms"
	@echo "  test     - build + run tests"
	@echo "  clean    - remove build artifacts"
	@echo ""
	@echo "Override OS/ARCH: make build OS=linux ARCH=arm64"
