#!/bin/bash

compileLinuxAMD64() {
  #cd container && podman build -t ghcr.io/memmaker/lfgo-amd64:latest . && podman push ghcr.io/memmaker/lfgo-amd64:latest && cd ..
  docker container run --rm --entrypoint='' \
      --arch=amd64 \
      -v ./src:/usr/src/app \
      -w /usr/src/app \
      ghcr.io/memmaker/lfgo-amd64:latest /bin/sh -c "CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags ebitensinglethread -o ./linux_amd64_ta -trimpath -ldflags '-s -w' ." && \
  mv ./src/linux_amd64_ta ./release/linux_amd64/ta
}

createZIP() {
    FILE="$1"
    zip -r "${FILE}.zip" "${FILE}"
    rm "${FILE}"
}


if [ -z "$1" ]; then
    echo "Must provide executable name"
    exit 1
fi

NAME="$1"
shift

echo "Creating release"
if [ -d "release" ]; then
    echo "warning: release dir exists. Removing all files!"
    rm -rf release/*
fi
mkdir -p release/linux_amd64
#mkdir -p release/linux_arm64
mkdir -p release/darwin_amd64
mkdir -p release/darwin_arm64
mkdir -p release/windows_amd64


compileLinuxAMD64
#compileLinuxARM64

cd src || exit
#CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags ebitensinglethread -o "../release/linux_amd64/${NAME}" -trimpath -ldflags '-s -w' "$@"
#CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -tags ebitensinglethread -o "../release/linux_arm64/${NAME}" -trimpath -ldflags '-s -w' "$@"

CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags ebitensinglethread -o "../release/darwin_amd64/${NAME}" -trimpath -ldflags '-s -w' "$@"
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build  -tags ebitensinglethread -o "../release/darwin_arm64/${NAME}" -trimpath -ldflags '-s -w' "$@"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -tags ebitensinglethread -o "../release/windows_amd64/${NAME}.exe" -trimpath -ldflags '-s -w' "$@"

# WASM
CGO_ENABLED=1 GOOS=js GOARCH=wasm go build -tags ebitensinglethread,web -o "../release/wasm/${NAME}.wasm" -trimpath -ldflags '-s -w' "$@"
cp ../wasm-template/* ../release/wasm/


cd ../release || exit
find .. -name '.DS_Store' -type f -delete
for SUBDIR in *
do
#    cp -r ../font "$SUBDIR"
#    cp -r ../data "$SUBDIR"
    cp ../readme.txt "$SUBDIR"
    cp ../changelog.txt "$SUBDIR"
    cp -r ../build/keydefs.txt "$SUBDIR"
    cp -r ../build/datafiles "$SUBDIR"
    createZIP "${SUBDIR}"
    rm -rf "$SUBDIR"
done

