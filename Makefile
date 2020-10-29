PROJECT_NAME := "webhookd"
PKG := "$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
.PHONY: all dep build clean test coverage coverhtml lint

all: build

lint: ## Lint the files
	@golint -set_exit_status ${PKG_LIST}

test: ## Run unittests
	@go test -short ${PKG_LIST}

race: dep ## Run data race detector
	@go test -race -short ${PKG_LIST}

msan: dep ## Run memory sanitizer
	@go test -msan -short ${PKG_LIST}

coverage: ## Generate global code coverage report
	./tools/coverage.sh;

coverhtml: ## Generate global code coverage report in HTML
	./tools/coverage.sh html;

dep: ## Get the dependencies
	@go mod download

build: dep ## Build the binary file
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/webhookd cmd/webhookd/main.go

build-win: dep ## Build the binary file
	@env CGO_ENABLED=0 GOARCH=386 GOOS=windows go build -o build/webhookd.exe cmd/webhookd/main.go

run: build
	@build/webhookd $(ARGS)

clean: ## Remove previous build
	@rm -f build/*

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
