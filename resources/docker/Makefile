# Makefile with examples for running gopherbot in a Docker container

.PHONY: prod dev clean

GOPHER_SOURCE_IMAGE?=lnxjedi/gopherbot:latest
GOPHER_BOTNAME?=$(notdir $(basename $(abspath .)))

# Example prod container that runs detached and restarts on failure.
prod:
	docker container run --name $(GOPHER_BOTNAME) --restart unless-stopped -d \
	  --log-driver journald --log-opt tag="$(GOPHER_BOTNAME)" \
	  --env-file environment -e HOSTNAME=$(HOSTNAME) \
	  $(GOPHER_SOURCE_IMAGE)

# A dev container that outputs directly to STDOUT/STDERR.
dev:
	docker container run --name $(GOPHER_BOTNAME) \
	  --env-file environment -e HOSTNAME=$(HOSTNAME) \
	  -e GOPHER_LOGLEVEL=debug --interactive --tty \
	  $(GOPHER_SOURCE_IMAGE)

# A container that mounts $(PWD) to the $(GOPHER_HOME), for local dev
# For the ubuntu/latest container, $(PWD) must be owned/writable by
# UID 1.
local:
	docker container run --name $(GOPHER_BOTNAME) \
	  -e HOSTNAME=$(HOSTNAME) -v $(PWD):/home/robot \
	  -e GOPHER_LOGLEVEL=debug --interactive --tty \
	  $(GOPHER_SOURCE_IMAGE)

# An interactive container for running the setup plugin
setup:
	docker container run --name floyd \
	  --interactive --tty -e HOSTNAME=$(HOSTNAME) \
	  $(GOPHER_SOURCE_IMAGE)

# A debug container that just executes a shell
debug:
	docker container run --name $(GOPHER_BOTNAME) \
	  --env-file environment -e HOSTNAME=$(HOSTNAME) \
	  -e GOPHER_LOGLEVEL=debug --interactive --tty \
	  --entrypoint /bin/bash \
	  $(GOPHER_SOURCE_IMAGE)

clean:
	docker container stop $(GOPHER_BOTNAME) || :
	docker container rm $(GOPHER_BOTNAME) || :
