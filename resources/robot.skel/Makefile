# Makefile with examples for running gopherbot

.PHONY: image dev prod clean allclean

include .env
export

# Build the robot's image, with the same name as the robot
image:
	docker image build -t $(GOPHER_BOTNAME) \
	--build-arg BOTNAME=$(GOPHER_BOTNAME) .

# A dev container that outputs the log to STDOUT.
dev:
	docker container run --name $(GOPHER_BOTNAME) \
	  --mount 'source=$(GOPHER_BOTNAME)-home,target=/home' \
	  $(GOPHER_BOTNAME)

# The secrets container can be used with 'encrypt' and
# 'store <task|repository> <secret|parameter>' to provide secrets
# to the robot that never get sent to Slack.
secrets:
	docker container run -it --name $(GOPHER_BOTNAME) \
	  --env-file setsecrets.env \
	  --mount 'source=$(GOPHER_BOTNAME)-home,target=/home' \
	  $(GOPHER_BOTNAME)

# Example prod container that runs detached and restarts on failure.
prod:
	docker container run --name $(GOPHER_BOTNAME) --restart on-failure:7 -d \
	  --log-driver journald --log-opt tag="$(GOPHER_BOTNAME)" \
	  --mount 'source=$(GOPHER_BOTNAME)-home,target=/home' \
	  $(GOPHER_BOTNAME)

clean:
	docker container stop $(GOPHER_BOTNAME) || :
	docker container rm $(GOPHER_BOTNAME) || :

allclean: clean
	docker image rm $(GOPHER_BOTNAME) || :
