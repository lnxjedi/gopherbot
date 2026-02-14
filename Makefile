# Makefile - just builds the binary, for dev mainly

.PHONY: clean test fulltest unit integration integration-full generate testbot static dist containers debug mcp

commit := -X main.Commit=$(shell git rev-parse --short HEAD)
version := $(shell ./get-version.sh)

TAR_ARCHIVE = gopherbot-linux-amd64.tar.gz
ZIP_ARCHIVE = gopherbot-linux-amd64.zip

GOOS ?= linux
CGO ?= 0
CTAG ?= latest

ifdef TEST
TESTARGS = -run ${TEST}
endif
ifeq ($(TEST),JSFull)
TESTARGS = -run TestJSFull
TESTENV = RUN_FULL=js
endif
ifneq ($(strip $(RUN_FULL)),)
TESTARGS = -run Test.*Full
TESTENV = RUN_FULL=$(RUN_FULL)
endif

static: gopherbot

gopherbot: main.go modules.go bot/* brains/*/* connectors/*/* gojobs/*/* goplugins/*/* history/*/* robot/* gotasks/*/* modules/*/*
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod readonly -ldflags "-s -w $(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot main.go modules.go

mcp: gopherbot-mcp

gopherbot-mcp: cmd/gopherbot-mcp/*.go
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod readonly -ldflags "-s -w $(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot-mcp ./cmd/gopherbot-mcp

debug:
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod readonly -ldflags "$(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot

clean:
	rm -f gopherbot gopherbot-mcp $(TAR_ARCHIVE) $(ZIP_ARCHIVE)

$(TAR_ARCHIVE): static
	./mkdist.sh

dist: $(TAR_ARCHIVE)

# Run test suite without coverage (see .gopherci/pipeline.sh)
# Full suites are opt-in: use RUN_FULL=js (or RUN_FULL=all) and only Test.*Full runs.
# Shortcut: TEST=JSFull make integration
unit:
	go test -mod readonly ./...

integration:
	${TESTENV} go test ${TESTARGS} -v --tags 'test integration netgo osusergo static_build' -mod readonly -race ./test

integration-full:
	RUN_FULL=all $(MAKE) integration

test: unit integration

fulltest: unit integration-full

# Generate Stringer methods
generate:
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./bot/
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./robot/

# Terminal robot that emits events gathered, for developing integration tests
testbot:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=amd64 go build -mod readonly -tags 'netgo osusergo static_build test' -o gopherbot
