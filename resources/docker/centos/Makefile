.PHONY: clean

image:
	docker image build -t lnxjedi/gopherbot:v2-centos -t lnxjedi/gopherbot:centos .

clean:
	docker image rm lnxjedi/gopherbot:centos || :
	docker image rm lnxjedi/gopherbot:v2-centos || :
