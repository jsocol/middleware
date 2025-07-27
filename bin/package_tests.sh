#!/bin/sh

modules=$(find . -type f -name go.mod)

for mod in $modules; do
    pkg=$(dirname "$mod")
    echo "testing $(basename "$pkg")"
    cd "$pkg" || exit 1
    go test -v ./...
    cd - || exit 1
done
