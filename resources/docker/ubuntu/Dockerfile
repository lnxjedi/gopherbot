FROM ubuntu:latest

RUN apt-get update && \
  apt-get -y upgrade && \
  apt-get install -y \
    curl \
    git \
    gcc \
    jq \
    make \
    openssh-client \
    python3 \
    ruby \
    unzip \
    zip && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/*

ARG username=daemon
ARG nonprivuser=nobody
ARG groupname=nogroup
# Common from here down
ARG installdir=/opt
ARG version=v2.0.0-snapshot
ARG goversion=1.13.5

ENV PATH=${PATH}:${HOME}/go/bin:/usr/local/go/bin
ENV HOME=/home/robot
ENV USER=${username}
ENV GROUP=${groupname}

RUN cd /usr/local && \
  curl -L https://dl.google.com/go/go${goversion}.linux-amd64.tar.gz | tar xzf - && \
  mkdir -p ${installdir} && \
  mkdir -p ${HOME} && \
  chown ${username}:${groupname} ${HOME} && \
  chmod 0750 ${HOME}

# Invalidate the cache when master changes
ADD https://api.github.com/repos/lnxjedi/gopherbot/commits/master /commit

## For dev
# ADD gopherbot-linux-amd64.tar.gz ${installdir}
# RUN chown ${nonprivuser}:${groupname} ${installdir}/gopherbot/gopherbot && \
#   chmod u+s ${installdir}/gopherbot/gopherbot

RUN cd ${installdir} && \
  curl -L https://github.com/lnxjedi/gopherbot/releases/download/${version}/gopherbot-linux-amd64.tar.gz | tar xzvf - && \
  chown ${nonprivuser}:${groupname} ${installdir}/gopherbot/gopherbot && \
  chmod u+s ${installdir}/gopherbot/gopherbot

WORKDIR ${HOME}
USER ${USER}:${GROUP}

ENTRYPOINT [ "/opt/gopherbot/gopherbot" , "-plainlog" ]
