# Makefile - just builds the binary, for dev mainly

.PHONY: clean test unit wireguard-plugin-test integration integration-build integration-run integration-mcp generate testbot static dist containers debug mcp docs-check

commit := -X main.Commit=$(shell git rev-parse --short HEAD)
version := $(shell ./get-version.sh)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
DIST_GOOS ?= linux
TAR_ARCHIVE = gopherbot-$(DIST_GOOS)-$(GOARCH).tar.gz
ZIP_ARCHIVE = gopherbot-$(DIST_GOOS)-$(GOARCH).zip

CGO ?= 0
CTAG ?= latest

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

# Run unit tests without coverage.
unit:
	go test -mod readonly ./...

wireguard-plugin-test: gopherbot
	./helpers/check-wireguard-plugin.sh

integration: integration-build
	@echo "Built ./gopherbot-integration"
	@echo "List suites: ./gopherbot-integration list-suites"
	@echo "Run a suite: ./gopherbot-integration run-suite TestBotName"

integration-build: gopherbot-integration

integration-run: gopherbot-integration
	./gopherbot-integration run-suite $(if $(TEST),$(TEST),all)

integration-mcp: gopherbot-mcp gopherbot-integration
	printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"run_integration_suite","arguments":{"suite":"$(if $(TEST),$(TEST),all)","build":false,"live":false,"include_output_tail":true,"tail_lines":80}}}' | ./gopherbot-mcp

test: unit wireguard-plugin-test integration

docs-check:
	./helpers/check-docs-hygiene.sh

# Generate Stringer methods
generate:
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./bot/
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./robot/

# Terminal robot that emits events gathered, for developing integration tests
testbot:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=amd64 go build -mod readonly -tags 'netgo osusergo static_build test' -o gopherbot
