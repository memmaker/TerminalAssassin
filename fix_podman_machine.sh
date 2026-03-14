#!/bin/zsh

podman machine stop && podman machine rm && podman machine init -v /Users/felix:/Users/felix && podman machine start