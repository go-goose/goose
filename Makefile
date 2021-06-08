.DEFAULT_GOAL := all

static-analysis:
	golangci-lint run -c .github/golangci-lint.config.yaml

test:
	go test -v ./...

all: static-analysis test
