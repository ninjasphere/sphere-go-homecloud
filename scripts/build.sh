#!/usr/bin/env bash
set -ex

OWNER=ninjasphere
BIN_NAME=sphere-go-homecloud
PROJECT_NAME=sphere-go-homecloud


# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

GIT_COMMIT="$(git rev-parse HEAD)"
GIT_DIRTY="$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)"
VERSION="$(grep "const Version " version.go | sed -E 's/.*"(.+)"$/\1/' )"

# remove working build
# rm -rf .gopath
if [ ! -d ".gopath" ]; then
	mkdir -p .gopath/src/github.com/${OWNER}
	ln -sf ../../../.. .gopath/src/github.com/${OWNER}/${PROJECT_NAME}
fi

export GOPATH="$(pwd)/.gopath"

# move the working path and build
cd .gopath/src/github.com/${OWNER}/${PROJECT_NAME}
go get -d -v ./...

# deal with juju/loggo change
GOOS= GOARCH= go get github.com/tools/godep
export PATH=$GOPATH/bin:$PATH
godep restore

# building the master branch on ci
if [ "$BUILDBOX_BRANCH" = "master" ]; then
	go build -ldflags "-X main.BugsnagKey ${BUGSNAG_KEY}" -tags release -o ./bin/${BIN_NAME}
else
	go build -o ./bin/${BIN_NAME}
fi
