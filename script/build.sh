#!/bin/bash

build=$(date +%FT%T%z)
version="$1"

ldflags="-s -w -X github.com/gerrittrigger/events/config.Build=$build -X github.com/gerrittrigger/events/config.Version=$version"
target="events"

go env -w GOPROXY=https://goproxy.cn,direct

# go tool dist list
GIN_MODE=release CGO_ENABLED=0 GOARCH=$(go env GOARCH) GOOS=$(go env GOOS) go build -ldflags "$ldflags" -o bin/$target main.go

upx bin/$target
