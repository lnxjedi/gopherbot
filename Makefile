# Makefile - just for Docker stuff, really

image:
	docker image build -t gopherbot:2.0.0 .

gopherbot:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'netgo osusergo static_build' -o gopherbot

# Example 'prod' container; note that the robot home directory is a
# persistent volume. You might want to use a different log driver for
# your environment, e.g. journald.
prod:
	docker container run --name gopherbot --restart unless-stopped -d \
	  --mount 'source=gopherbot-home,target=/opt/robot' \
	  --env-file $(HOME)/gopherbot.env gopherbot:2.0.0

# Example 'dev' container bind mounts the binary and all scripts and
# libraries from the local repository.
dev: gopherbot
	docker container run --name gopherbot --restart unless-stopped -d \
	  --mount 'source=gopherbot-home,target=/opt/robot' \
	  --mount "type=bind,source=$(PWD)/gopherbot,target=/opt/gopherbot/gopherbot" \
	  --mount "type=bind,source=$(PWD)/lib,target=/opt/gopherbot/lib" \
	  --mount "type=bind,source=$(PWD)/plugins,target=/opt/gopherbot/plugins" \
	  --mount "type=bind,source=$(PWD)/tasks,target=/opt/gopherbot/tasks" \
	  --mount "type=bind,source=$(PWD)/jobs,target=/opt/gopherbot/jobs" \
	  --mount "type=bind,source=$(PWD)/scripts,target=/opt/gopherbot/scripts" \
	  --env-file $(HOME)/gopherbot.env gopherbot:2.0.0

clean:
	rm -f gopherbot
	docker container stop gopherbot || :
	docker container rm gopherbot || :

allclean: clean
	docker image rm gopherbot:2.0.0
