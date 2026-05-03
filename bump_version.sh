#!/bin/zsh

REPO="$(dirname "$0")"
PART="$1"

if [[ "$PART" != "major" && "$PART" != "minor" && "$PART" != "patch" ]]; then
    echo "Usage: $0 [major|minor|patch]"
    exit 1
fi

if ! git -C "$REPO" diff --quiet || ! git -C "$REPO" diff --cached --quiet; then
    read "reply?Working tree has uncommitted changes. Commit them now? [y/N] "
    if [[ "$reply" != "y" && "$reply" != "Y" ]]; then
        echo "Aborting."
        exit 1
    fi
    read "msg?Commit message: "
    git -C "$REPO" add -A && git -C "$REPO" commit -m "$msg"
fi

current=$(git -C "$REPO" describe --tags --abbrev=0 2>/dev/null | tr -d 'v\n')
if [[ -z "$current" ]]; then
    current="0.0.0"
fi
major=$(echo "$current" | cut -d. -f1)
minor=$(echo "$current" | cut -d. -f2)
patch=$(echo "$current" | cut -d. -f3)

case "$PART" in
    major) major=$((major + 1)); minor=0; patch=0 ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    patch) patch=$((patch + 1)) ;;
esac

new="v${major}.${minor}.${patch}"
echo "Version bumped: v${current} -> ${new}"

read "reply?Tag current git state as ${new}? [y/N] "
if [[ "$reply" == "y" || "$reply" == "Y" ]]; then
    git -C "$REPO" tag "$new" && echo "Tagged: ${new}"
fi

read "reply?Push current state to remote? [y/N] "
if [[ "$reply" == "y" || "$reply" == "Y" ]]; then
    git -C "$REPO" push --tags && echo "Pushed."
fi







