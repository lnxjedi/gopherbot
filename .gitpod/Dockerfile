FROM ubuntu:latest

ARG goversion=1.12
ENV HOME=/home/gitpod
ENV PATH=${PATH}:${HOME}/go/bin:/usr/local/go/bin

RUN apt-get update && \
  apt-get -y upgrade && \
  apt-get install -y \
    curl \
    git \
    gcc \
    jq \
    openssh-client \
    python \
    python-yaml \
    ruby \
    zip \
    unzip && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* && \
  cd /usr/local && \
  curl -L https://dl.google.com/go/go${goversion}.linux-amd64.tar.gz | tar xzf -
