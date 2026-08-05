APP=skeleton
DEV_BIN=bin/app
PORT?=9528
ENV?=dev

.PHONY: build
build:
	@go build -o releases/${APP} ./cmd/app
	@go build -o releases/${APP}-cli ./cmd/cli

# Agent 终端下未签名 Go 二进制无法访问局域网；增量 build + ad-hoc 签名后启动。
# 源码未改时 go build 几乎秒回，不必每次全量重编。
.PHONY: run
run:
	@mkdir -p bin
	@go build -o ${DEV_BIN} ./cmd/app
	@codesign -s - --force ${DEV_BIN} >/dev/null 2>&1
	@exec ./${DEV_BIN} -e=${ENV} -p=${PORT}

.PHONY: windows
windows:
	@GOARCH=amd64 GOOS=windows go build -ldflags="-s" -o releases/${APP}-win ./cmd/app
	@GOARCH=amd64 GOOS=windows go build -ldflags="-s" -o releases/${APP}-win-cli ./cmd/cli

.PHONY: linux
linux:
	@GOARCH=amd64 GOOS=linux go build -ldflags="-s" -o releases/${APP}-linux ./cmd/app
	@GOARCH=amd64 GOOS=linux go build -ldflags="-s" -o releases/${APP}-linux-cli ./cmd/cli

.PHONY: darwin
darwin:
	@GOARCH=amd64 GOOS=darwin go build -ldflags="-s" -o releases/${APP}-darwin ./cmd/app
	@GOARCH=amd64 GOOS=darwin go build -ldflags="-s" -o releases/${APP}-darwin-cli ./cmd/cli

.PHONY: lint
lint:
	@if ! command -v gofumpt &> /dev/null; then \
		echo "gofumpt not found, installing..."; \
		go install mvdan.cc/gofumpt@latest; \
	fi
	@gofumpt -l -w .

.PHONY: generate
generate:
	@go generate -x

.PHONY: clean
clean:
	@go clean -i .
	@rm -rf releases bin

.PHONY: docs
docs:
	@if ! command -v swag &> /dev/null; then \
		echo "swag not found, installing..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@swag init -g main.go -d ./cmd/app,./internal/app/controller,./pkg/response --exclude "docs/,vendor/,**/*_test.go" --parseDepth 3 --parseDependency --parseInternal -o ./docs

.PHONY: help
help:
	@echo "1. make build - [go build]"
	@echo "2. make run - [incremental build + codesign + start, Agent/局域网可用]"
	@echo "3. make windows - [make window package]"
	@echo "4. make linux - [make linux package]"
	@echo "5. make darwin - [make darwin package]"
	@echo "6. make lint - [gofumpt -l -w .]"
	@echo "7. make generate - [go generate -x]"
	@echo "8. make clean - [remove releases/bin and cached files]"
	@echo "9. make docs - [generate swagger docs]"
	@echo "   PORT=9528 ENV=dev make run"
