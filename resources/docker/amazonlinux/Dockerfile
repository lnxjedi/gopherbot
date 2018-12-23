FROM amazonlinux:latest

RUN yum -y update && \
  yum -y install \
    git \
    jq \
    openssh-clients \
    shadow-utils \
    tar \
    PyYAML && \
  amazon-linux-extras install ruby2.4 && \
  yum clean all && \
  rm -rf /var/cache/yum

# Common from here down
ARG installdir=/opt/gopherbot
ARG version=v2.0.0-snapshot
ARG username=robot
ARG groupname=robot
ARG uid=49
ARG gid=49
ARG executioncontext=/home/runcontext

ENV GOPHER_DOCKER=true
ENV HOME=/home/${username}
ENV USER=${username}
ENV GROUP=${groupname}
ENV CONTEXT=${executioncontext}

# Invalidate the cache when master changes
ADD https://api.github.com/repos/lnxjedi/gopherbot/commits/master /commit

RUN groupadd -g ${gid} ${groupname} && \
  useradd -m -u ${uid} -g ${groupname} -d ${HOME} ${username} && \
  mkdir -p ${installdir} && \
  cd ${installdir} && \
  curl -L https://github.com/lnxjedi/gopherbot/releases/download/${version}/gopherbot-linux-amd64.tar.gz | tar xzvf - && \
  mkdir -p ${executioncontext}/protected/custom \
    ${executioncontext}/protected/brain \
    ${HOME}/workspace \
    ${HOME}/history && \
  chown ${username}:bin /opt/gopherbot/gopherbot && \
  chmod 4550 /opt/gopherbot/gopherbot && \
  chown ${username}:${groupname} ${executioncontext}/protected/custom \
    ${executioncontext}/protected/brain \
    ${HOME}/workspace \
    ${HOME}/history && \
  chmod 550 ${executioncontext} && \
  chown bin:root ${executioncontext}

WORKDIR ${executioncontext}/protected
USER bin:bin

ENTRYPOINT [ "/opt/gopherbot/gopherbot" , "-plainlog" ]
