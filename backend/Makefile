GO=go
GOTEST=$(GO) test
BIN_DIR=bin
EXPORT_RESULT?=FALSE
.DEFAULT_GOAL:=help

##################################### Binary #####################################

.PHONY: pre
pre: ## Create the bin directory
	@mkdir -p $(BIN_DIR)

.PHONY: clean
clean: ## Clean the bin directory
	@rm -rf $(BIN_DIR)
	@rm -f ./coverage.out

.PHONY: build
build: pre ## Create the main binary
	@CGO_ENABLED=1 $(GO) build -ldflags="-s -w" -o $(BIN_DIR)/sloggo main.go

##################################### Beautify #####################################

.PHONY: format
format: ## Format the source code
	@find . -name '*.go' -not -path './vendor/*' | xargs -n1 go fmt

.PHONY: lint
lint:
	@$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2
	@golangci-lint run --timeout 3m0s

##################################### Test #####################################

.PHONY: goconvey
goconvey: ## Run goconvey
	@$(GO) install github.com/smartystreets/goconvey@latest
	@goconvey -port 1234

.PHONY: test
test: ## Run the tests
ifeq ($(EXPORT_RESULT), TRUE)
	@$(GO) install github.com/jstemmer/go-junit-report/v2@latest
	@$(eval OUTPUT_OPTIONS = | go-junit-report -iocopy -set-exit-code -out junit-report.xml)
endif
	@$(GO) clean -testcache
	@$(GOTEST) -v -race ./... $(OUTPUT_OPTIONS)

.PHONY: coverage
coverage: $(wildcard **/**/*.go) ## Run the tests and generate the coverage reports
	@$(GOTEST) -cover -covermode=count -coverprofile=coverage.out ./...
ifeq ($(EXPORT_RESULT), TRUE)
	@$(GO) install github.com/axw/gocov/gocov@latest
	@$(GO) install github.com/AlekSi/gocov-xml@latest
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@gocov convert coverage.out | gocov-xml > coverage.xml
	@rm -f coverage.out
endif

.PHONY: vulnerability-scan ## Scan the project for known vulnerabilities
vulnerability-scan:
	@$(GO) install github.com/google/osv-scanner/v2/cmd/osv-scanner@latest
	@osv-scanner scan ./

##################################### Help #####################################

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
