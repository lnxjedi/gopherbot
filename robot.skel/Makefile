# Makefile with examples for running gopherbot

.PHONY: prod image dev clean allclean

include .env

GOPHER_SOURCE_IMAGE?=lnxjedi/gopherbot:latest
GOPHER_BUILD_IMAGE?=$(GOPHER_SOURCE_IMAGE)

# Example prod container that runs detached and restarts on failure.
prod:
	docker container run --name $(GOPHER_BOTNAME) --restart on-failure:7 -d \
	  --log-driver journald --log-opt tag="$(GOPHER_BOTNAME)" \
	  --mount 'source=$(GOPHER_BOTNAME)-home,target=/home' \
	  --env-file .env -e HOSTNAME=$(HOSTNAME) \
	  $(GOPHER_BUILD_IMAGE)

# Build a custom image
image:
	[ $(GOPHER_BUILD_IMAGE) != $(GOPHER_SOURCE_IMAGE) ] || ( echo "Custom build image not defined"; exit 1 )
	docker image build -t $(GOPHER_BUILD_IMAGE) \
	--build-arg SOURCE_IMAGE=$(GOPHER_SOURCE_IMAGE) .

# A dev container that outputs the log to STDOUT.
dev:
	docker container run --name $(GOPHER_BOTNAME) \
	  --mount 'source=$(GOPHER_BOTNAME)-home,target=/home' \
	  --env-file .env -e HOSTNAME=$(HOSTNAME) \
	  $(GOPHER_BUILD_IMAGE)

# The secrets container can be used with 'encrypt' and
# 'store <task|repository> <secret|parameter>' to provide secrets
# to the robot that never get sent to Slack.
secrets:
	docker container run -it --name $(GOPHER_BOTNAME) \
	  --env-file setsecrets.env \
	  --mount 'source=$(GOPHER_BOTNAME)-home,target=/home' \
	  --env-file .env -e HOSTNAME=$(HOSTNAME) \
	  $(GOPHER_BUILD_IMAGE)

clean:
	docker container stop $(GOPHER_BOTNAME) || :
	docker container rm $(GOPHER_BOTNAME) || :

allclean: clean
	docker image rm $(GOPHER_BOTNAME) || :
