#!/bin/bash
set -e
mkdir -p build
cd src
go build -o ../build/terminal-assassin .
