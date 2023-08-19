#!/bin/zsh

cd src || exit
# WASM
CGO_ENABLED=1 GOOS=js GOARCH=wasm go build -tags ebitensinglethread,web -o "../release/wasm/ta.wasm" -trimpath -ldflags '-s -w' "."
cp ../wasm-template/* ../release/wasm/
