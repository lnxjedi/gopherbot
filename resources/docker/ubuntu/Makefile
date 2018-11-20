# Makefile - just for Docker stuff, really

image:
	docker image build -t lnxjedi/gopherbot:2.0.0-ubuntu -t lnxjedi/gopherbot:ubuntu .

clean:
	docker image rm lnxjedi/gopherbot:ubuntu || :
	docker image rm lnxjedi/gopherbot:2.0.0-ubuntu || :
