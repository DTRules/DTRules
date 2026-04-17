# DTRules Go Makefile
#
# Build with version information:
#   make build
#
# Install:
#   make install

# Version info
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Build flags
PKG := github.com/DTRules/DTRules/pkg/dtrules/version
LDFLAGS := -s -w \
           -X $(PKG).Version=$(VERSION) \
           -X $(PKG).Commit=$(COMMIT) \
           -X $(PKG).Date=$(DATE) \
           -X $(PKG).GitCommit=$(COMMIT) \
           -X $(PKG).GitBranch=$(GIT_BRANCH) \
           -X $(PKG).BuildDate=$(DATE)

# Output
BINARY   := dtrules
BUILD_DIR := build
DIST_DIR  := dist

.PHONY: all build install clean test version release

all: build

build:
	@echo "Building $(BINARY) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/dtrules/

install:
	@echo "Installing $(BINARY) $(VERSION)..."
	go install -ldflags "$(LDFLAGS)" ./cmd/dtrules/

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	go clean

test:
	@echo "Running tests..."
	go test ./...

version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Branch:  $(GIT_BRANCH)"
	@echo "Date:    $(DATE)"

# Release: cross-compile for all platforms and produce checksums in dist/
release:
	@echo "Building release $(VERSION)..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-linux-amd64       ./cmd/dtrules/
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-linux-arm64       ./cmd/dtrules/
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-darwin-amd64      ./cmd/dtrules/
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-darwin-arm64      ./cmd/dtrules/
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe ./cmd/dtrules/
	cd $(DIST_DIR) && sha256sum * > checksums.txt
	@echo "Release artifacts in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/

# Cross-compilation targets (individual platforms, output to build/)
.PHONY: build-linux build-darwin build-windows build-all

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/dtrules/

build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/dtrules/
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/dtrules/

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/dtrules/

build-all: build-linux build-darwin build-windows
	@echo "Built all platforms"
