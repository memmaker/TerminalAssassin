#!/bin/zsh

compileLinuxAMD64() {
  #cd container && podman build -t ghcr.io/memmaker/lfgo-amd64:latest . && podman push ghcr.io/memmaker/lfgo-amd64:latest && cd ..
  docker container run --rm --entrypoint='' \
      --arch=amd64 \
      -v ./src:/usr/src/app \
      -w /usr/src/app \
      ghcr.io/memmaker/lfgo-amd64:latest /bin/sh -c "CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags ebitensinglethread -o ./linux_amd64_ta -trimpath -ldflags '-s -w' ." && \
  mv ./src/linux_amd64_ta ./release/linux_amd64/ta
}

mkdir -p release/linux_amd64

compileLinuxAMD64