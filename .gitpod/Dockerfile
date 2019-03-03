FROM amazonlinux:latest

ARG goversion=1.12
ENV HOME=/home/gitpod
ENV PATH=${PATH}:${HOME}/go/bin:/usr/local/go/bin

RUN yum -y update && \
  yum -y install \
    gcc \
    zip \
    git \
    jq \
    openssh-clients \
    shadow-utils \
    tar \
    PyYAML && \
  amazon-linux-extras install ruby2.4 && \
  yum clean all && \
  rm -rf /var/cache/yum && \
  cd /usr/local && \
  curl -L https://dl.google.com/go/go${goversion}.linux-amd64.tar.gz | tar xzf -
