FROM golang:1.13.4

RUN apt-get update && \
  apt-get -y upgrade && \
  apt-get install -y \
    curl \
    git \
    jq \
    openssh-client \
    python \
    python-yaml \
    ruby \
    zip \
    unzip && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* && \
  echo -e "export PATH=$PATH:/usr/local/go/bin:/workspace/go/bin" > /etc/profile.d/golang.sh

ENV GOPATH=/workspace/golang