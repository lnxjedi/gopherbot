# Makefile - just builds the binary, for dev mainly

.PHONY: clean test generate testbot static modular dist containers debug

commit := $(shell git rev-parse --short HEAD)
version := $(shell ./get-version.sh)

MODULES = goplugins/knock.so goplugins/duo.so goplugins/meme.so goplugins/totp.so \
	connectors/slack.so connectors/rocket.so brains/dynamodb.so history/file.so

TAR_ARCHIVE = gopherbot-linux-amd64.tar.gz
ZIP_ARCHIVE = gopherbot-linux-amd64.zip

GOOS ?= linux
CGO ?= 0
CTAG ?= latest

modular: CGO = 1
modular: BUILDTAG = modular
modular: gopherbot $(MODULES)

ifdef TEST
TESTARGS = -run ${TEST}
endif

static: gopherbot

gopherbot: main*.go bot/* brains/*/* connectors/*/* goplugins/*/* history/*/*
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w -X main.Commit=$(commit) -X main.Version=$(version)" -tags "netgo osusergo static_build $(BUILDTAG)" -o gopherbot

debug:
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-X main.Commit=$(commit) -X main.Version=$(version)" -tags "netgo osusergo static_build" -o gopherbot

# modules
connectors/slack.so: connectors/slack-mod.go connectors/slack/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w" -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

connectors/rocket.so: connectors/rocket-mod.go connectors/rocket/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w" -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

history/file.so: history/file-mod.go history/file/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w" -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

goplugins/duo.so: goplugins/duo-mod.go goplugins/duo/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w" -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

goplugins/knock.so: goplugins/knock-mod.go goplugins/knock/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w" -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

goplugins/meme.so: goplugins/meme-mod.go goplugins/meme/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w" -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

goplugins/totp.so: goplugins/totp-mod.go goplugins/totp/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w" -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

brains/dynamodb.so: brains/dynamodb-mod.go brains/dynamodb/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-s -w" -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

clean:
	rm -f gopherbot $(MODULES) $(TAR_ARCHIVE) $(ZIP_ARCHIVE)

$(TAR_ARCHIVE): modular
	./.gopherci/mkdist.sh

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
