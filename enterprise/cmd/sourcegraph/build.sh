#!/usr/bin/env bash

# This script builds the single binary (Sourcegraph App).

unset CDPATH
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
set -eu

pkg="github.com/sourcegraph/sourcegraph/enterprise/cmd/sourcegraph"
binary="${OUT-.bin/$(basename ${pkg})$(go env GOEXE)}"
ldflags=("-X github.com/sourcegraph/sourcegraph/internal/version.version=${VERSION-0.0.0+dev}")
ldflags+=("-X github.com/sourcegraph/sourcegraph/internal/version.timestamp=$(date +%s)")
ldflags+=("-X github.com/sourcegraph/sourcegraph/internal/conf/deploy.forceType=single-program")

# TTTTTTTTTT ENTERPRISE=1 DEV_WEB_BUILDER=esbuild yarn run build-web
go build -trimpath \
   -ldflags "${ldflags[*]}" \
   -buildmode exe \
   -tags dist \
   -o "${binary}" \
   "${pkg}"

echo $binary
