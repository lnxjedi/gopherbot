# Makefile with examples for running gopherbot in a Docker container

.PHONY: prod dev clean

include environment

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
	  -e GOPHER_LOGLEVEL=debug \
	  $(GOPHER_SOURCE_IMAGE)

# A debug container that just executes a shell
debug:
	docker container run --name $(GOPHER_BOTNAME) \
	  --env-file environment -e HOSTNAME=$(HOSTNAME) \
	  -e GOPHER_LOGLEVEL=debug -it --entrypoint /bin/bash \
	  $(GOPHER_SOURCE_IMAGE)

clean:
	docker container stop $(GOPHER_BOTNAME) || :
	docker container rm $(GOPHER_BOTNAME) || :
