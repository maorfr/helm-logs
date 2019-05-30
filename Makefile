HELM_HOME ?= $(shell helm home)
HELM_PLUGIN_DIR ?= $(HELM_HOME)/plugins/helm-logs
VERSION := $(shell sed -n -e 's/version:[ "]*\([^"]*\).*/\1/p' plugin.yaml)
DIST := $(CURDIR)/_dist
LDFLAGS := "-w -X main.version=${VERSION}"

export GO111MODULE=on

.PHONY: install
install: bootstrap build
	cp helmlogs $(HELM_PLUGIN_DIR)
	cp plugin.yaml $(HELM_PLUGIN_DIR)

.PHONY: hookInstall
hookInstall: bootstrap build

.PHONY: build
build:
	go build -o helmlogs -ldflags $(LDFLAGS) ./main.go

.PHONY: dist
dist:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build -o helmlogs -ldflags $(LDFLAGS) ./main.go
	tar -zcvf $(DIST)/helm-logs-linux-$(VERSION).tgz helmlogs README.md LICENSE.txt plugin.yaml
	GOOS=darwin GOARCH=amd64 go build -o helmlogs -ldflags $(LDFLAGS) ./main.go
	tar -zcvf $(DIST)/helm-logs-macos-$(VERSION).tgz helmlogs README.md LICENSE.txt plugin.yaml
	GOOS=windows GOARCH=amd64 go build -o helmlogs.exe -ldflags $(LDFLAGS) ./main.go
	tar -zcvf $(DIST)/helm-logs-windows-$(VERSION).tgz helmlogs.exe README.md LICENSE.txt plugin.yaml

.PHONY: bootstrap
bootstrap:
	go mod download
