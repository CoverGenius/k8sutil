#!/usr/bin/env bash
set -e

VERSION=$(git rev-parse --short HEAD)
DISTDIR="artifacts/"
export GO111MODULE=on

for pair in linux/amd64 darwin/amd64 windows/amd64; do
    GOOS=`echo $pair | cut -d'/' -f1`
    GOARCH=`echo $pair | cut -d'/' -f2`
    OBJECT_FILE="k8sutil-$VERSION-$GOOS-$GOARCH"
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$DISTDIR/$OBJECT_FILE"
done

