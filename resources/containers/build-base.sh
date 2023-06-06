#!/bin/bash

docker pull docker.io/amazonlinux:2023
docker build -f containerfile.base -t ghcr.io/lnxjedi/gopherbot-base:latest .
