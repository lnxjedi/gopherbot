# Makefile - just builds the binary, for dev mainly

.PHONY: testbot

commit := $(shell git rev-parse --short HEAD)

gopherbot: main.go bot/* brains/*/* connectors/*/* goplugins/*/* history/*/*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -ldflags "-X main.Commit=$(commit)" -tags 'netgo osusergo static_build' -o gopherbot

testbot:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -tags 'netgo osusergo static_build test' -o gopherbot
