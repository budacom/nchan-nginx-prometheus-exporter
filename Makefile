VERSION ?= dev
REVISION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

.PHONY: test build

test:
	go test ./...

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/prometheus/common/version.Version=$(VERSION) -X github.com/prometheus/common/version.Revision=$(REVISION)" -o nchan-prometheus-exporter .
