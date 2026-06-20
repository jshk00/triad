.PHONY: test coverage lint

# If we're not debugging the Makefile, don't echo recipes.
MAKEFLAGS += -s
# We don't need make's built-in rules.
MAKEFLAGS += --no-builtin-rules
# Be pedantic about undefined variables.
MAKEFLAGS += --warn-undefined-variables

# It's necessary to set this because some environments don't link sh -> bash.
SHELL := /usr/bin/env bash -o errexit -o pipefail -o nounset

GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
COVER_PROFILE := "triad.cov"
COVER_HTML := "coverage.html"
MIN_COVERAGE ?= 80

lint:
	golangci-lint run --verbose

test:
	go test \
		-v=test2json \
		-cover \
		-coverprofile=$(COVER_PROFILE) \
		-count=1 \
		-json \
		> test-output.txt || (jq . < test-output.txt; false)

coverage:
	go tool cover -html=$(COVER_PROFILE) -o $(COVER_HTML)
	COVERAGE="$$(go tool cover -func $(COVER_PROFILE))"; \
		echo "$$COVERAGE"; \
		TEST_COVERAGE="$$(echo "$$COVERAGE" | grep 'total:' | awk '{print substr($$3, 1, length($$3)-1)}')"; \
		printf "\nUnit-tests passed with $$TEST_COVERAGE%% coverage\n"; \
		if [[ 1 -eq "$$(echo $$TEST_COVERAGE $(MIN_COVERAGE) | awk '{if ($$1 < $$2) print 1}')" ]]; then \
			echo "Minimum $(MIN_COVERAGE)% coverage is not met"; \
			exit 1; \
		fi

clean-test-data:
	rm -rf test-output.txt $(COVER_PROFILE) $(COVER_HTML)
