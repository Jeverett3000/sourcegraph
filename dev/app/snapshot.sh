#!/usr/bin/env bash

docker run --rm --privileged \
       -v $(pwd):/go/src/github.com/sourcegraph/sourcegraph \
       -v /var/run/docker.sock:/var/run/docker.sock \
       -w /go/src/github.com/sourcegraph/sourcegraph \
       neilotoole/xcgo:latest goreleaser --snapshot --rm-dist
