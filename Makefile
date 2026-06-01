GOPATH = $(shell go env GOPATH)
GOBIN = $(GOPATH)/bin

.PHONY: test coverage lint

test:
	@go test -v -count 1 ./... -coverprofile test.cov

coverage:
	@go tool cover -html=test.cov -o coverage.html

lint:
	@golangci-lint run --verbose
