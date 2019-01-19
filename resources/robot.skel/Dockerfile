FROM lnxjedi/gopherbot:latest

ARG BOTNAME

USER root:root
## Use hostname liberally to provide information about where the robot
## is running, supplied by the 'info' command.
ENV HOSTNAME=${BOTNAME}.container

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

COPY .env ${HOME}

RUN chown bin:bin .env && \
  chmod 0400 .env

USER ${USER}:${GROUP}

# Uncomment for debugging start-up issues
#ENTRYPOINT []
