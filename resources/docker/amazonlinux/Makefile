.PHONY: image clean

image:
	docker image build -t lnxjedi/gopherbot:v2-amazon -t lnxjedi/gopherbot:amazon .

clean:
	docker image rm lnxjedi/gopherbot:amazon || :
	docker image rm lnxjedi/gopherbot:v2-amazon || :
