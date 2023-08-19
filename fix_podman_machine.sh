#!/bin/zsh

podman machine rm && podman machine init -v /Users/felix:/Users/felix && podman machine start