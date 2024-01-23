
BINARY = kubectl-copy_secret
GOARCH = amd64
GOOS = linux

VERSION?=?
COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

GITHUB_USERNAME=
LDFLAGS = -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT} -X main.BRANCH=${BRANCH}"

.PHONY: build
build:
	go build -o bin/kubectl-copy_secret cmd/kubectl-copy_secret.go