#!/usr/bin/env bash
# Build the browser version and publish it to https://ruzzoli.de/games/terminal-assassin/play/
set -euo pipefail
cd "$(dirname "$0")"
./build_web.sh
rsync -az --delete /Users/felix/Projects/fx-games/site/terminal-assassin/play/ ruzzoli.de:/var/www/ruzzoli.de/games/terminal-assassin/play/
echo "published: https://ruzzoli.de/games/terminal-assassin/play/"
