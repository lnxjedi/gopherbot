FROM golang:1.13.8

RUN apt-get update && \
  apt-get -y upgrade && \
  apt-get install -y \
    curl \
    git \
    jq \
    less \
    openssh-client \
    python3 \
    ruby \
    zip \
    unzip && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* && \
  echo "export PATH=$PATH:/usr/local/go/bin:/workspace/go/bin" > /etc/profile.d/golang.sh

ENV GOPATH=/workspace/golang
