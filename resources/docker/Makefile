# Makefile with examples for running gopherbot
installdir := $(shell dirname $(shell dirname $(PWD)))

# Example 'prod' container; note that the robot home directory is a
# persistent volume. You might want to use a different log driver for
# your environment, e.g. journald.
prod:
	docker container run --name gopherbot --restart unless-stopped -d \
	  --mount 'source=gopherbot-home,target=/home/robot' \
	  --env-file $(HOME)/gopherbot.env gopherbot:2.0.0

# Example 'dev' container bind mounts the local install dir for running
# gopherbot in a container.
dev:
	docker container run --name gopherbot \
	  --mount 'source=gopherbot-home,target=/home/robot' \
	  --mount "type=bind,source=$(installdir),target=/opt/gopherbot" \
	  --env-file $(HOME)/gopherbot-dev.env lnxjedi/gopherbot:amazon

clean:
	docker container stop gopherbot || :
	docker container rm gopherbot || :
