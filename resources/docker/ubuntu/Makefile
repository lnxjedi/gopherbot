.PHONY: clean

image:
	docker image build -t lnxjedi/gopherbot:v2-ubuntu -t lnxjedi/gopherbot:ubuntu -t lnxjedi/gopherbot:latest .

clean:
	docker image rm lnxjedi/gopherbot:ubuntu || :
	docker image rm lnxjedi/gopherbot:v2-ubuntu || :
	docker image rm lnxjedi/gopherbot:latest || :
