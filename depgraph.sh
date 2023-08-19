#!/bin/zsh
cd src || exit
godepgraph -p github.com/hajimehoshi,github.com/tinne26,golang.org -s -novendor .  | dot -Tpng -o godepgraph.png
open godepgraph.png
cd ..