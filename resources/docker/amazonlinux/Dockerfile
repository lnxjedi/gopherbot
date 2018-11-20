FROM amazonlinux:latest

ARG installdir=/opt/gopherbot
ARG version=v2.0.0-snapshot
ARG username=robot
ARG groupname=robot
ARG uid=49
ARG gid=49

WORKDIR ${installdir}

ENV HOME=/home/robot

RUN yum -y update && \
  yum -y install \
    git \
    jq \
    openssh-clients \
    shadow-utils \
    unzip \
    PyYAML \
    && \
  amazon-linux-extras install ruby2.4 && \
  groupadd -g ${gid} ${groupname} && \
  useradd -m -u ${uid} -g ${groupname} -d ${HOME} robot && \
  yum clean all && \
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

ADD https://github.com/lnxjedi/gopherbot/releases/download/${version}/gopherbot-linux-amd64.zip ${installdir}

RUN unzip gopherbot-linux-amd64.zip && \
  rm gopherbot-linux-amd64.zip

WORKDIR ${installdir}/env

RUN chmod 770 . && \
  chown ${username}:${groupname} /opt/gopherbot/gopherbot && \
  chmod u+s /opt/gopherbot/gopherbot

ENTRYPOINT [ "/opt/gopherbot/gopherbot" , "-plainlog" ]
