PROJECT         := CNAB.ARM-Converter
FILENAME        := cnabtoarmtemplate
ORG             := simongdavies
BINDIR          := $(CURDIR)/bin
GOFLAGS         :=
LDFLAGS         := -w -s
TESTFLAGS       := -v
GO = GO111MODULE=on go
COMMIT ?= $(shell git rev-parse --short HEAD)

ifeq ($(OS),Windows_NT)
	TARGET = $(FILENAME).exe
	SHELL  = cmd /c
	CHECK  = where
else
	TARGET = $(FILENAME)
	SHELL  ?= bash
	CHECK  ?= which
endif

GIT_TAG   := $(shell git describe --tags --always)
VERSION   ?= ${GIT_TAG}
LDFLAGS   += -X  github.com/$(ORG)/$(PROJECT)/pkg.Version=$(VERSION) -X github.com/$(ORG)/$(PROJECT)/pkg.Commit=$(COMMIT)

.PHONY: default
default: build

.PHONY: build
build:

	$(GO) mod tidy
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINDIR)/$(TARGET) github.com/$(ORG)/$(PROJECT)/cmd/...

CX_OSES  = linux windows darwin
CX_ARCHS = amd64

.PHONY: build-release
build-release:
ifeq ($(OS),Windows_NT)
	powershell -executionPolicy bypass -NoLogo -NoProfile -File ./build/build-release.ps1 -oses '$(CX_OSES)' -arch  $(CX_ARCHS) -ldflags $(LDFLAGS) -filename $(FILENAME) -project $(PROJECT) -bindir $(BINDIR) -org $(ORG)
else
	@for os in $(CX_OSES); do \
		echo "building $$os"; \
		for arch in $(CX_ARCHS); do \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build -ldflags '$(LDFLAGS)' -o $(BINDIR)/$(TARGET)-$$os-$$arch github.com/$(ORG)/$(PROJECT)/cmd/...; \
		done; \
		if [ $$os = 'windows' ]; then \
			mv $(BINDIR)/$(TARGET)-$$os-$$arch $(BINDIR)/$(TARGET)-$$os-$$arch.exe; \
		fi; \
	done
endif

.PHONY: debug
debug:
	$(GO) build $(GOFLAGS) -o $(BINDIR)/$(TARGET) github.com/$(ORG)/$(PROJECT)/cmd/...

.PHONY: test
test:
	$(GO) test $(TESTFLAGS) ./...

.PHONY: lint
lint:
	golangci-lint run --config ./golangci.yml

HAS_GOLANGCI     := $(shell $(CHECK) golangci-lint)
HAS_GOIMPORTS    := $(shell $(CHECK) goimports)
GOLANGCI_VERSION := v1.30.0

.PHONY: bootstrap
bootstrap:

ifndef HAS_GOLANGCI
ifeq ($(OS),Windows_NT)
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
else
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_VERSION)
endif
endif

ifeq ($(OS),Windows_NT)
	choco install diffutils -y
endif