FROM golang:1.11.1

ARG installdir=/opt/gopherbot
ARG version=v2.0.0-snapshot
ARG username=robot
ARG groupname=robot
ARG uid=49
ARG gid=49

WORKDIR ${installdir}

ENV HOME=/home/robot

COPY docker/ ${installdir}

RUN groupadd -g ${gid} ${groupname} && \
  useradd -m -u ${uid} -g ${groupname} -d ${HOME} robot && \
  apt-get update && \
  apt-get -y upgrade && \
  apt-get install -y \
    jq \
    ruby \
    unzip \
    && \
  apt-get clean && \
  wget https://github.com/lnxjedi/gopherbot/releases/download/${version}/gopherbot-linux-amd64.zip && \
  unzip gopherbot-linux-amd64.zip && \
  rm gopherbot-linux-amd64.zip && \
  mkdir -p ${HOME}/conf \
    ${HOME}/brain \
    ${HOME}/workspace \
    ${HOME}/history \
    && \
  chown ${username}:${groupname} ${HOME}/conf \
    ${HOME}/brain \
    ${HOME}/workspace \
    ${HOME}/history

USER ${username}:${groupname}

ENTRYPOINT [ "./gopherbot" , "-plainlog", "-c", "${HOME}/conf" ]
