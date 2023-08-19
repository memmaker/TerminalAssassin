#!/bin/zsh

VER=$(semver get beta)

echo "Pushing release $VER"

# for every zip file in /release

for file in ./release/*.zip
do
  if test "${file#*windows}" != "$file"
  then
    butler push "$file" memmakerx/terminal-assassin:windows-amd-beta --userversion "$VER"
    continue
  elif test "${file#*darwin_amd}" != "$file"
  then
    butler push "$file" memmakerx/terminal-assassin:osx-amd-beta --userversion "$VER"
    continue
  elif test "${file#*darwin_arm}" != "$file"
    then
      butler push "$file" memmakerx/terminal-assassin:osx-arm-beta --userversion "$VER"
      continue
  elif test "${file#*linux}" != "$file"
  then
    butler push "$file" memmakerx/terminal-assassin:linux-amd-beta --userversion "$VER"
    continue
  elif test "${file#*wasm}" != "$file"
  then
    butler push "$file" memmakerx/terminal-assassin:html-beta --userversion "$VER"
    continue
  fi
done
