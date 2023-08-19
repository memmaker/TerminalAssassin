#!/bin/zsh

podman build -t ghcr.io/memmaker/lfgo-amd64:latest . && podman push ghcr.io/memmaker/lfgo-amd64:latest