#!/bin/sh

go tool dist list | sed 's/\// /g' | awk '{ printf("GOOS=%s GOARCH=%s go build -o viz-%s-%s\n", $1, $2, $1, $2) }' | sh
