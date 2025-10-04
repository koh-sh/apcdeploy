.PHONY: test fmt cov tidy lint build modernize modernize-fix ci tool-install

COVFILE = coverage.out
COVHTML = cover.html

test:
	go test ./... -json | go tool tparse -all

fmt:
	go tool gofumpt -l -w .

cov:
	go test -cover ./... -coverprofile=$(COVFILE)
	go tool cover -html=$(COVFILE) -o $(COVHTML)
	rm $(COVFILE)

tidy:
	go mod tidy -v

lint:
	go tool golangci-lint run -v

build:
	go build

ci: fmt modernize-fix lint test build

# Go Modernize
modernize:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./...

modernize-fix:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./...

tool-install:
	go get -tool mvdan.cc/gofumpt
	go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint
	go get -tool github.com/mfridman/tparse
	go get -tool github.com/spf13/cobra-cli
