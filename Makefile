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

gopherbot: main*.go bot/* brains/*/* connectors/*/* goplugins/*/* history/*/*
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w $(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot main.go main_static.go

debug:
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "$(commit) $(version)" -tags "netgo osusergo static_build" -o gopherbot

clean:
	rm -f gopherbot $(TAR_ARCHIVE) $(ZIP_ARCHIVE)

$(TAR_ARCHIVE): static
	./mkdist.sh

dist: $(TAR_ARCHIVE)

containers:
	buildah pull quay.io/lnxjedi/gopherbot-base:latest
	buildah pull quay.io/lnxjedi/gopherbot-base-theia:latest
	# NOTE: set BUILDREF in the environment to build anything other than default branch
	buildah bud --build-arg buildref=${BUILDREF} -f resources/containers/minimal/Containerfile -t quay.io/lnxjedi/gopherbot:$(CTAG) ./resources/containers/minimal/
	buildah bud --build-arg buildref=${BUILDREF} -f resources/containers/theia/Containerfile -t quay.io/lnxjedi/gopherbot-theia:$(CTAG) ./resources/containers/theia/
	buildah bud --build-arg buildref=${BUILDREF} -f resources/containers/dev/Containerfile -t quay.io/lnxjedi/gopherbot-dev:$(CTAG) ./resources/containers/dev/

# Run test suite without coverage (see .gopherci/pipeline.sh)
test:
	go test ${TESTARGS} -v --tags 'test integration netgo osusergo static_build' -mod vendor -race ./test

# Generate Stringer methods
generate:
	go generate -v --tags 'test integration netgo osusergo static_build' -mod vendor ./bot/
	go generate -v --tags 'test integration netgo osusergo static_build' -mod vendor ./robot/

# Terminal robot that emits events gathered, for developing integration tests
testbot:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -tags 'netgo osusergo static_build test' -o gopherbot
