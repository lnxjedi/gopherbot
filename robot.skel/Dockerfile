## NOTE: OUTDATED - this note will be removed when Docker support
## rework is done.

ARG SOURCE_IMAGE=lnxjedi/gopherbot:latest
FROM ${SOURCE_IMAGE}

USER root:root

## Example customisation section installs gcc, zip
## and Go - needed to build Gopherbot.
#ARG goversion=1.11.4
#ENV PATH=${PATH}:${HOME}/go/bin:/usr/local/go/bin

#RUN yum -y install \
#    gcc \
#    zip && \
#  yum clean all && \
#  rm -rf /var/cache/yum && \
#  cd /usr/local && \
#  curl -L https://dl.google.com/go/go${goversion}.linux-amd64.tar.gz | tar xzf -

## /end Customisation

USER ${USER}:${GROUP}

# Uncomment for debugging start-up issues
#ENTRYPOINT []
