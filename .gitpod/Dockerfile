FROM golang:1.12.5

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
  mkdir -p /workspace/go && \
  GOPATH=/workspace/go go get -u golang.org/x/tools/cmd/gopls
