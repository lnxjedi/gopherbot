# Makefile - just builds the binary, for dev mainly

# .PHONY: gopherbot

gopherbot: main.go bot/* brains/* connectors/* goplugins/* history/* vendor/*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -tags 'netgo osusergo static_build' -o gopherbot
