# Makefile - just builds the binary, for dev mainly

.PHONY: clean test generate testbot static dist containers debug

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

static: gopherbot

gopherbot: main.go modules.go bot/* brains/*/* connectors/*/* gojobs/*/* goplugins/*/* history/*/* robot/* gotasks/*/* modules/*/*
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod readonly -ldflags "-s -w $(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot main.go modules.go

debug:
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod readonly -ldflags "$(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot

clean:
	rm -f gopherbot $(TAR_ARCHIVE) $(ZIP_ARCHIVE)

$(TAR_ARCHIVE): static
	./mkdist.sh

dist: $(TAR_ARCHIVE)

unittest:
	go test ./...

# Run test suite without coverage (see .gopherci/pipeline.sh)
test:
	GOPHER_IDE= go test ${TESTARGS} -v --tags 'test integration netgo osusergo static_build' -mod readonly -race ./test

test-all: unittest test

# Generate Stringer methods
generate:
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./bot/
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./robot/

# Terminal robot that emits events gathered, for developing integration tests
testbot:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=amd64 go build -mod readonly -tags 'netgo osusergo static_build test' -o gopherbot
