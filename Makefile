# Makefile - just builds the binary, for dev mainly

.PHONY: clean test fulltest unit integration integration-build integration-run integration-mcp integration-legacy integration-full generate testbot static dist containers debug mcp docs-check

commit := -X main.Commit=$(shell git rev-parse --short HEAD)
version := $(shell ./get-version.sh)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
DIST_GOOS ?= linux
TAR_ARCHIVE = gopherbot-$(DIST_GOOS)-$(GOARCH).tar.gz
ZIP_ARCHIVE = gopherbot-$(DIST_GOOS)-$(GOARCH).zip

CGO ?= 0
CTAG ?= latest

ifdef TEST
TESTARGS = -run ${TEST}
endif
ifeq ($(TEST),JSFull)
TESTARGS = -run TestJSFull
TESTENV = RUN_FULL=js
endif
ifeq ($(TEST),LuaFull)
TESTARGS = -run TestLuaFull
TESTENV = RUN_FULL=lua
endif
ifeq ($(TEST),ShFull)
TESTARGS = -run TestShFull
TESTENV = RUN_FULL=sh
endif
ifeq ($(TEST),GoFull)
TESTARGS = -run TestGoFull
TESTENV = RUN_FULL=go
endif
ifneq ($(strip $(RUN_FULL)),)
TESTARGS = -run Test.*Full
TESTENV = RUN_FULL=$(RUN_FULL)
endif

static: gopherbot

gopherbot: main.go modules.go bot/* brains/*/* connectors/*/* gojobs/*/* goplugins/*/* history/*/* robot/* gotasks/*/* modules/*/*
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=${GOARCH} go build -mod readonly -ldflags "-s -w $(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot main.go modules.go

mcp: gopherbot-mcp

gopherbot-mcp: cmd/gopherbot-mcp/*.go
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=${GOARCH} go build -mod readonly -ldflags "-s -w $(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot-mcp ./cmd/gopherbot-mcp

gopherbot-integration: cmd/gopherbot-integration/*.go integration/suites/*.go bot/* brains/*/* connectors/*/* gojobs/*/* goplugins/*/* history/*/* robot/* gotasks/*/* modules/*/*
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=${GOARCH} go build -mod readonly -ldflags "-s -w $(commit) $(version)" -tags "test integration netgo osusergo static_build" -o gopherbot-integration ./cmd/gopherbot-integration

debug:
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=${GOARCH} go build -mod readonly -ldflags "$(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot

clean:
	rm -f gopherbot gopherbot-mcp gopherbot-integration $(TAR_ARCHIVE) $(ZIP_ARCHIVE)

$(TAR_ARCHIVE): GOOS=$(DIST_GOOS)
$(TAR_ARCHIVE): static
	GOOS=${DIST_GOOS} GOARCH=${GOARCH} ./mkdist.sh

dist: $(TAR_ARCHIVE)

# Run test suite without coverage (see .gopherci/pipeline.sh)
# Full suites are opt-in: use RUN_FULL=js (or RUN_FULL=all) and only Test.*Full runs.
# Shortcut: TEST=JSFull make integration
unit:
	go test -mod readonly ./...

integration: integration-build
	@echo "Built ./gopherbot-integration"
	@echo "List suites: ./gopherbot-integration list-suites"
	@echo "Run a suite: ./gopherbot-integration run-suite TestBotName"
	@echo "Legacy go test harness: make integration-legacy"

integration-build: gopherbot-integration

integration-run: gopherbot-integration
	./gopherbot-integration run-suite $(if $(TEST),$(TEST),all)

integration-mcp: gopherbot-mcp gopherbot-integration
	printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"run_integration_suite","arguments":{"suite":"$(if $(TEST),$(TEST),all)","build":false,"live":false,"include_output_tail":true,"tail_lines":80}}}' | ./gopherbot-mcp

integration-legacy: gopherbot
	${TESTENV} go test ${TESTARGS} -v --tags 'test integration netgo osusergo static_build' -mod readonly -race ./test

integration-full:
	RUN_FULL=all $(MAKE) integration-legacy

test: unit integration

fulltest: unit integration-full

docs-check:
	./helpers/check-docs-hygiene.sh

# Generate Stringer methods
generate:
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./bot/
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./robot/

# Terminal robot that emits events gathered, for developing integration tests
testbot:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=amd64 go build -mod readonly -tags 'netgo osusergo static_build test' -o gopherbot
