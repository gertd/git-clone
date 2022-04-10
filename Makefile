# Makefile
NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
ATTN_COLOR=\033[33;01m
#

.PHONY: all

all: deps gen build lint test

deps:
	@echo "$(WARN_COLOR)==> deps $(NO_COLOR)"
	@go install gotest.tools/gotestsum@v1.7.0
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.2
	@go install github.com/goreleaser/goreleaser@v1.6.3

gen:
	@echo "$(ATTN_COLOR)==> generate$(NO_COLOR)"
	@go generate ./... 

build:
	@echo "$(WARN_COLOR)==> build $(NO_COLOR)"
	@goreleaser build --single-target --snapshot --rm-dist

clean: 
	@echo "$(WARN_COLOR)==> clean $(NO_COLOR)"
	@rm -rf ./dist

lint:
	@echo "$(ATTN_COLOR)==> Lint $(NO_COLOR)"
	@golangci-lint run

test:
	@echo "$(WARN_COLOR)==> test $(NO_COLOR)"
	@gotestsum ./...

release: 	
	@echo "$(WARN_COLOR)==> release $(NO_COLOR)"
	@goreleaser release --rm-dist
