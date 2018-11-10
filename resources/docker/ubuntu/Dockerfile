FROM ubuntu:latest

ARG installdir=/opt/gopherbot
ARG version=v2.0.0-snapshot
ARG username=robot
ARG groupname=robot
ARG uid=49
ARG gid=49
ARG commit=434237b

WORKDIR ${installdir}

ENV HOME=/home/robot

RUN groupadd -g ${gid} ${groupname} && \
  useradd -m -u ${uid} -g ${groupname} -d ${HOME} robot && \
  apt-get update && \
  apt-get -y upgrade && \
  apt-get install -y \
    curl \
    git \
    jq \
    openssh-client \
    python \
    python-yaml \
    ruby \
    unzip \
    && \
  apt-get clean && \
  mkdir -p ${HOME}/conf \
    ${HOME}/brain \
    ${HOME}/workspace \
    ${HOME}/history \
    ${HOME}/.gopherbot \
    && \
  chown ${username}:${groupname} ${HOME}/conf \
    ${HOME}/brain \
    ${HOME}/workspace \
    ${HOME}/history \
    ${HOME}/.gopherbot

USER ${username}:${groupname}

RUN   curl -L -o gopherbot-linux-amd64.zip https://github.com/lnxjedi/gopherbot/releases/download/${version}/gopherbot-linux-amd64.zip?commit=${commit} && \
  unzip gopherbot-linux-amd64.zip && \
  rm gopherbot-linux-amd64.zip

ENTRYPOINT [ "./gopherbot" , "-plainlog" ]
