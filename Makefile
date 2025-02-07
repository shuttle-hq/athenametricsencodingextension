SHELL = /bin/bash
.SHELLFLAGS = -o pipefail -c

# SRC_ROOT is the top of the source tree.
SRC_ROOT := $(shell git rev-parse --show-toplevel)

# build tags required by any component should be defined as an independent variables and later added to GO_BUILD_TAGS below
GO_BUILD_TAGS=""
# These ldflags allow the build tool to omit the symbol table, debug information, and the DWARF symbol table to downscale binary size.
GO_BUILD_LDFLAGS="-s -w"
GOTEST_TIMEOUT?= 600s
GOTEST_OPT?= -race -timeout $(GOTEST_TIMEOUT) -parallel 4 --tags=$(GO_BUILD_TAGS)
GOCMD?= go
GOOS=$(shell $(GOCMD) env GOOS)
GOARCH=$(shell $(GOCMD) env GOARCH)
GOTESTARCH?=$(GOARCH)

TOOLS_MOD_DIR    := $(SRC_ROOT)/internal/tools
TOOLS_MOD_REGEX  := "\s+_\s+\".*\""
TOOLS_PKG_NAMES  := $(shell grep -E $(TOOLS_MOD_REGEX) < $(TOOLS_MOD_DIR)/tools.go | tr -d " _\"")
TOOLS_BIN_DIR    := $(SRC_ROOT)/.tools
TOOLS_BIN_NAMES  := $(addprefix $(TOOLS_BIN_DIR)/, $(notdir $(TOOLS_PKG_NAMES)))

.PHONY: install-tools
install-tools: $(TOOLS_BIN_NAMES)

$(TOOLS_BIN_DIR):
	mkdir -p $@

$(TOOLS_BIN_NAMES): $(TOOLS_BIN_DIR) $(TOOLS_MOD_DIR)/go.mod
	cd $(TOOLS_MOD_DIR) && GOOS="" GOARCH="" $(GOCMD) build -o $@ -trimpath $(filter %/$(notdir $@),$(TOOLS_PKG_NAMES))

ADDLICENSE          := $(TOOLS_BIN_DIR)/addlicense
LINT                := $(TOOLS_BIN_DIR)/golangci-lint
GOIMPORTS           := $(TOOLS_BIN_DIR)/goimports
PORTO               := $(TOOLS_BIN_DIR)/porto
GOFUMPT             := $(TOOLS_BIN_DIR)/gofumpt
GOVULNCHECK         := $(TOOLS_BIN_DIR)/govulncheck
GCI                 := $(TOOLS_BIN_DIR)/gci
GOTESTSUM           := $(TOOLS_BIN_DIR)/gotestsum

GOTESTSUM_OPT?= --rerun-fails=1
ADDLICENSE_CMD := $(ADDLICENSE) -s=only -y "" -c "Shuttle"

ALL_PKG_DIRS := $(shell $(GOCMD) list -f '{{ .Dir }}' ./... | sort)
ALL_SRC_AND_DOC := $(shell find $(ALL_PKG_DIRS) -name "*.md" -o -name "*.go" -o -name "*.yaml" -not -path '*/third_party/*' -type f | sort)

# BUILD_TYPE should be one of (dev, release).
BUILD_TYPE?=release

.DEFAULT_GOAL := all

.PHONY: all
all: install-tools lint goporto test

.PHONY: test
test: $(GOTESTSUM)
	$(GOTESTSUM) $(GOTESTSUM_OPT) --packages="./..." -- $(GOTEST_OPT)

.PHONY: addlicense
addlicense: $(ADDLICENSE)
	@ADDLICENSEOUT=$$(for f in $$($(ALL_SRC_AND_SHELL)); do \
		`$(ADDLICENSE_CMD) "$$f" 2>&1`; \
	done); \
	if [ "$$ADDLICENSEOUT" ]; then \
		echo "$(ADDLICENSE) FAILED => add License errors:\n"; \
		echo "$$ADDLICENSEOUT\n"; \
		exit 1; \
	else \
		echo "Add License finished successfully"; \
	fi

.PHONY: checklicense
checklicense: $(ADDLICENSE)
	@licRes=$$(for f in $$($(ALL_SRC_AND_SHELL)); do \
		awk '/Copyright Shuttle|generated|GENERATED/ && NR<=3 { found=1; next } END { if (!found) print FILENAME }' $$f; \
		awk '/SPDX-License-Identifier: Apache-2.0|generated|GENERATED/ && NR<=4 { found=1; next } END { if (!found) print FILENAME }' $$f; \
		$(ADDLICENSE_CMD) -check "$$f" 2>&1 || echo "$$f"; \
		done); \
	if [ -n "$${licRes}" ]; then \
		echo "license header checking failed:"; echo "$${licRes}"; \
		exit 1; \
	else \
		echo "Check License finished successfully"; \
	fi

.PHONY: fmt
fmt: $(GOFUMPT) $(GOIMPORTS)
	$(GOFUMPT) -l -w .
	$(GOIMPORTS) -w -local github.com/shuttle-hq/athenametricsencodingextension ./

.PHONY: lint
lint: $(LINT) checklicense
	$(LINT) run --allow-parallel-runners --build-tags integration --timeout=30m --path-prefix $(shell basename "$(CURDIR)")

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK)
	$(GOVULNCHECK) ./...

.PHONY: goporto
goporto: $(PORTO)
	$(PORTO) -w --include-internal ./

.PHONY: tidy
tidy:
	rm -fr go.sum
	$(GOCMD) mod tidy -compat=1.22.0

.PHONY: toolchain
toolchain:
	$(GOCMD) get toolchain@none

.PHONY: moddownload
moddownload:
	$(GOCMD) mod download

.PHONY: gci
gci: $(GCI)
	@echo "running $(GCI)"
	@$(GCI) write -s standard -s default -s "prefix(github.com/shuttle-hq/athenametricsencodingextension)" $(ALL_SRC_AND_DOC)

.PHONY: generate
generate: install-tools
	PATH="$(TOOLS_BIN_DIR):$$PATH" $(GOCMD) generate ./...
	$(MAKE) fmt

.PHONY: clean
clean:
	@echo "Removing the built tools"
	rm -rf $(TOOLS_BIN_DIR)

.PHONY: checks
checks:
	$(MAKE) -j4 goporto
	$(MAKE) -j4 tidy
	$(MAKE) -j4 generate
	@git diff --exit-code || (echo 'Some files need committing' && git status && exit 1)
