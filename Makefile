# Makefile - just builds the binaries, for dev mainly

.PHONY: clean test generate testbot static dist containers debug

# Variables for versioning and build information
commit := -X main.Commit=$(shell git rev-parse --short HEAD)
version := $(shell ./get-version.sh)

# Archive filenames
TAR_ARCHIVE = gopherbot-linux-amd64.tar.gz
ZIP_ARCHIVE = gopherbot-linux-amd64.zip

# Build environment variables with default values
GOOS ?= linux
CGO ?= 0
CTAG ?= latest

# Conditional test arguments
ifdef TEST
TESTARGS = -run ${TEST}
endif

# Default target: build both gopherbot and privsep
static: gopherbot privsep

# Target to build the main Gopherbot binary
gopherbot: main.go modules.go bot/* brains/*/* connectors/*/* gojobs/*/* goplugins/*/* history/*/* robot/* gotasks/*/* modules/*/*

	@echo "Building gopherbot..."
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod readonly \
		-ldflags "-s -w $(commit) $(version)" \
		-tags "netgo osusergo static_build" \
		-o gopherbot main.go modules.go

# Target to build the privsep helper binary
privsep: helpers/privsep/privSepHelper.go

	@echo "Building privsep helper..."
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod readonly \
		-ldflags "-s -w $(commit) $(version)" \
		-tags "netgo osusergo static_build" \
		-o privsep helpers/privsep/privSepHelper.go

# Debug build for Gopherbot (does not build privsep)
debug:
	@echo "Building gopherbot in debug mode..."
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod readonly \
		-ldflags "$(commit) $(version)" \
		-tags "netgo osusergo static_build" \
		-o gopherbot

# Clean up generated binaries and archives
clean:
	@echo "Cleaning up binaries and archives..."
	rm -f gopherbot privsep $(TAR_ARCHIVE) $(ZIP_ARCHIVE)

# Create the tar archive after building static binaries
$(TAR_ARCHIVE): static
	@echo "Creating tar archive..."
	./mkdist.sh

# Distribution target depends on the tar archive
dist: $(TAR_ARCHIVE)

# Run test suite without coverage (see .gopherci/pipeline.sh)
test:
	@echo "Running tests..."
	go test ${TESTARGS} -v --tags 'test integration netgo osusergo static_build' -mod readonly -race ./test

# Generate Stringer methods and other generated code
generate:
	@echo "Generating code for bot package..."
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./bot/
	@echo "Generating code for robot package..."
	go generate -v --tags 'test integration netgo osusergo static_build' -mod readonly ./robot/

# Terminal robot that emits events gathered, for developing integration tests
testbot:
	@echo "Building testbot version of gopherbot..."
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=amd64 go build -mod readonly \
		-tags 'netgo osusergo static_build test' \
		-o gopherbot

