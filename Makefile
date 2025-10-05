.PHONY: test fmt cov tidy lint lint-fix build modernize modernize-fix ci tool-install e2e-setup e2e-run e2e-clean e2e-full

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

lint-fix:
	go tool golangci-lint run -v --fix

build:
	go build

ci: fmt modernize-fix lint-fix test build

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

e2e-setup:
	cd e2e/terraform && terraform init && terraform apply

e2e-run:
	./e2e/e2e-test.sh

e2e-clean:
	cd e2e/terraform && ./cleanup.sh && terraform destroy

e2e-full: e2e-setup e2e-run e2e-clean
