FROM golang:1.14.4

ARG installdir=/opt
ARG version=v2.0.0-beta3
ARG nonprivuser=nobody

ENV PATH=${PATH}:/opt/gopherbot
ENV GOPATH=/workspace/golang

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
  echo "export PATH=$PATH:/usr/local/go/bin:/workspace/go/bin" > /etc/profile.d/golang.sh && \
  cd ${installdir} && \
  curl -L https://github.com/lnxjedi/gopherbot/releases/download/${version}/gopherbot-linux-amd64.tar.gz | tar xzvf - && \
  chown ${nonprivuser} ${installdir}/gopherbot/gopherbot && \
  chmod u+s ${installdir}/gopherbot/gopherbot

WORKDIR /home
ENTRYPOINT [ "/bin/bash" ]
