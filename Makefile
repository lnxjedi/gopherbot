# Makefile - just builds the binary, for dev mainly

.PHONY: clean test generate testbot modular static

commit := $(shell git rev-parse --short HEAD)

MODULES = goplugins/knock.so goplugins/duo.so goplugins/meme.so goplugins/totp.so \
	connectors/slack.so connectors/rocket.so connectors/terminal.so brains/dynamodb.so

GOOS ?= linux
CGO ?= 0

modular: CGO = 1
modular: BUILDTAG = modular
modular: gopherbot $(MODULES)

static: gopherbot

gopherbot: main.go bot/* brains/*/* connectors/*/* goplugins/*/* history/*/*
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-X main.Commit=$(commit)" -tags "netgo osusergo static_build $(BUILDTAG)" -o gopherbot

# modules
connectors/slack.so: connectors/slack-mod.go connectors/slack/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

connectors/rocket.so: connectors/rocket-mod.go connectors/rocket/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

connectors/terminal.so: connectors/terminal-mod.go connectors/terminal/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

goplugins/duo.so: goplugins/duo-mod.go goplugins/duo/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

goplugins/knock.so: goplugins/knock-mod.go goplugins/knock/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

goplugins/meme.so: goplugins/meme-mod.go goplugins/meme/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

goplugins/totp.so: goplugins/totp-mod.go goplugins/totp/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

brains/dynamodb.so: brains/dynamodb-mod.go brains/dynamodb/*.go
	GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -o $@ -buildmode=plugin -tags 'netgo osusergo static_build module' $<

clean:
	rm -f gopherbot $(MODULES)

# Run test suite
test:
	go test -v --tags 'test integration netgo osusergo static_build' -mod vendor -cover -race -coverprofile coverage.out -coverpkg ./... ./bot

# Generate Stringer methods
generate:
	go generate -v --tags 'test integration netgo osusergo static_build' -mod vendor ./bot/

# Terminal robot that emits events gathered, for developing integration tests
testbot:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -tags 'netgo osusergo static_build test' -o gopherbot
