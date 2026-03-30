# DTRules Go Makefile
#
# Build with version information:
#   make build
#
# Install:
#   make install

# Version info
# Uses git describe to get version from tags (e.g., "v1.0.0" or "v1.0.0-5-gabcdef")
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Check if working directory is clean (empty = clean, non-empty = dirty)
GIT_DIRTY := $(shell git status --porcelain 2>/dev/null)

# If dirty, omit commit hash (code doesn't match any commit)
ifdef GIT_DIRTY
  GIT_COMMIT ?= uncommitted
  VERSION ?= $(GIT_BRANCH)-dev
else
  GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
  VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
endif

# Build flags
PKG := github.com/DTRules/DTRules/pkg/dtrules/version
LDFLAGS := -X $(PKG).Version=$(VERSION)
LDFLAGS += -X $(PKG).GitCommit=$(GIT_COMMIT)
LDFLAGS += -X $(PKG).GitBranch=$(GIT_BRANCH)
LDFLAGS += -X $(PKG).BuildDate=$(BUILD_DATE)

# Output
BINARY := dtrules
BUILD_DIR := build

.PHONY: all build install clean test version

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
	rm -rf $(BUILD_DIR)
	go clean

test:
	@echo "Running tests..."
	go test ./...

version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(GIT_COMMIT)"
	@echo "Branch:  $(GIT_BRANCH)"
	@echo "Date:    $(BUILD_DATE)"

# Cross-compilation targets
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
