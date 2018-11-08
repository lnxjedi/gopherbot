# Makefile - just for Docker stuff, really

image:
	docker image build -t lnxjedi/gopherbot:2.0.0-amazon -t lnxjedi/gopherbot:amazon .

clean:
	docker image rm lnxjedi/gopherbot-amazon:2.0.0
	docker image rm lnxjedi/gopherbot:latest
