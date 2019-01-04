FROM ubuntu:latest

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
    unzip && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/*

# Common from here down
ARG installdir=/opt/gopherbot
ARG version=v2.0.0-snapshot
ARG username=robot
ARG groupname=robot
ARG uid=49
ARG gid=49

# The GOPHER_DOCKER var alters defaults in distributed gopherbot.yaml
ENV GOPHER_DOCKER=true
ENV HOME=/home/${username}
ENV USER=${username}
ENV GROUP=${groupname}

# Invalidate the cache when master changes
ADD https://api.github.com/repos/lnxjedi/gopherbot/commits/master /commit

RUN groupadd -g ${gid} ${groupname} && \
  useradd -m -u ${uid} -g ${groupname} -d ${HOME} ${username} && \
  mkdir -p ${installdir} && \
  cd ${installdir} && \
  curl -L https://github.com/lnxjedi/gopherbot/releases/download/${version}/gopherbot-linux-amd64.tar.gz | tar xzvf - && \
  mkdir -p ${HOME}/custom \
    ${HOME}/workspace \
    ${HOME}/brain \
    ${HOME}/history && \
  chown bin:${groupname} /opt/gopherbot/gopherbot \
    ${HOME} && \
  chmod 4450 /opt/gopherbot/gopherbot && \
  chmod 0570 ${HOME} && \
  chown ${username}:${groupname} ${HOME}/custom \
    ${HOME}/workspace && \
  chown bin:bin ${HOME}/brain \
    ${HOME}/history && \
  chmod 700 ${HOME}/brain

WORKDIR ${HOME}
USER ${USER}:${GROUP}

ENTRYPOINT [ "/opt/gopherbot/gopherbot" , "-plainlog" ]
