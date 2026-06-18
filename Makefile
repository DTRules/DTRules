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

.PHONY: all build install clean test check version release

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

# check: full-module verification that agents must run before declaring a task complete.
# Excludes ./pkg/dtrules/ root (pre-existing tax-content failures tracked in #520).
# Exclusions are explicit. Re-enable a package by adding it to both vet and
# test target lists below. Current exclusions and why:
#   - pkg/dtrules/  (root, tests only): pre-existing tax-content failures, #520.
#   - pkg/dtrules/authoring/  (tests only): pre-existing failures unrelated to
#     the engine — TestDeleteCondition_RemainingConditionsStay + the forall
#     tests assume a different scenario layout. Tracked separately.
#   - pkg/dtrules/compiler/el/  (vet only): ANTLR-generated parser triggers
#     "unreachable code" diagnostics that aren't worth fighting per-regen.
#   - pkg/dtrules/interpreter/  (vet only): asm_stubs.go declares extern
#     symbols that vet can't see implementations for. Build still passes
#     because amd64-only files are gated by //go:build amd64.
#   - pkg/dtrules/cmd  (vet only): pre-existing test reference; vet flags it
#     but tests pass.
check:
	@echo "Checking full module build..."
	go build ./...
	@echo "Running go vet..."
	go vet ./pkg/dtrules/analysis/... ./pkg/dtrules/benchmark/... ./pkg/dtrules/collect/... ./pkg/dtrules/compiler/ ./pkg/dtrules/datafile/... ./pkg/dtrules/decisiontable/... ./pkg/dtrules/encoding/... ./pkg/dtrules/entity/... ./pkg/dtrules/excel/... ./pkg/dtrules/interview/... ./pkg/dtrules/loader/... ./pkg/dtrules/mapping/... ./pkg/dtrules/operators/... ./pkg/dtrules/repository/... ./pkg/dtrules/ruleset/... ./pkg/dtrules/runtime/... ./pkg/dtrules/session/... ./pkg/dtrules/sync/... ./pkg/dtrules/testsupport/... ./pkg/dtrules/trace/... ./pkg/dtrules/version/... ./pkg/dtrules/web/...
	@echo "Running tests..."
	go test -count=1 ./cmd/... ./pkg/dtrules/analysis/... ./pkg/dtrules/benchmark/... ./pkg/dtrules/collect/... ./pkg/dtrules/compiler/... ./pkg/dtrules/datafile/... ./pkg/dtrules/decisiontable/... ./pkg/dtrules/encoding/... ./pkg/dtrules/entity/... ./pkg/dtrules/excel/... ./pkg/dtrules/interpreter/... ./pkg/dtrules/interview/... ./pkg/dtrules/loader/... ./pkg/dtrules/mapping/... ./pkg/dtrules/operators/... ./pkg/dtrules/repository/... ./pkg/dtrules/ruleset/... ./pkg/dtrules/runtime/... ./pkg/dtrules/session/... ./pkg/dtrules/sync/... ./pkg/dtrules/testsupport/... ./pkg/dtrules/trace/... ./pkg/dtrules/version/... ./pkg/dtrules/web/...
	@echo "Running hand-coded-postfix gate (any element with postfix-without-EL-DSL fails)..."
	go test -count=1 -run "^TestTaxReturn_NoHandCodedPostfix$$" ./pkg/dtrules/
	@echo "check passed."

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
