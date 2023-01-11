#!/usr/bin/env bash

ROOTDIR="$(realpath $(dirname "${BASH_SOURCE[0]}")/../..)"
GORELEASER_CROSS_VERSION=v1.19.5

# TTTTTTTTTT TODO(sqs): unskip ENTERPRISE=1 DEV_WEB_BUILDER=esbuild yarn run build-web

exec docker run --rm --privileged \
       -v "$ROOTDIR":/go/src/github.com/sourcegraph/sourcegraph \
       -v /var/run/docker.sock:/var/run/docker.sock \
       #-e "GITHUB_TOKEN=$GITHUB_TOKEN" \
       #-e "DOCKER_USERNAME=$DOCKER_USERNAME" -e "DOCKER_PASSWORD=$DOCKER_PASSWORD" -e "DOCKER_REGISTRY=$DOCKER_REGISTRY" \
       -w /go/src/github.com/sourcegraph/sourcegraph \
       goreleaser/goreleaser-cross:${GORELEASER_CROSS_VERSION} \
       --config dev/app/goreleaser.yaml --rm-dist "$@"
