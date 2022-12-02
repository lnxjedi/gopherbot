#!/bin/bash

for IMAGE in minimal theia dev
do
    docker build -f $IMAGE/Containerfile -t quay.io/lnxjedi/gopherbot-$IMAGE:latest $IMAGE/
done

