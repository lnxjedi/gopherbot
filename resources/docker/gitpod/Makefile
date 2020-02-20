.PHONY: clean

image:
	docker image build -t lnxjedi/gopherbot:v2-gitpod -t lnxjedi/gopherbot:gitpod .

clean:
	docker image rm lnxjedi/gopherbot:gitpod || :
	docker image rm lnxjedi/gopherbot:v2-gitpod || :
