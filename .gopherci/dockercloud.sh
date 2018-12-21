#!/bin/bash -e

# Trigger the automatic build 
curl -X POST https://cloud.docker.com/$DOCKER_TRIGGER > /dev/null

echo "Docker Cloud build triggered"
