.PHONY: test fmt cov cov-check tidy lint lint-fix build modernize modernize-fix ci tool-install e2e-setup e2e-run e2e-clean e2e-full

COVFILE = coverage.out
COVHTML = cover.html
GITHUB_REPOSITORY = koh-sh/apcdeploy

test:
	go test ./... -json | go tool tparse -all

fmt:
	go tool gofumpt -l -w .

cov:
	go test -cover ./... -coverprofile=$(COVFILE)
	go tool cover -html=$(COVFILE) -o $(COVHTML)
	rm $(COVFILE)

cov-check:
	go test -cover ./... -coverprofile=$(COVFILE)
	CI=1 GITHUB_REPOSITORY=$(GITHUB_REPOSITORY) octocov
	rm $(COVFILE)

tidy:
	go mod tidy -v

lint:
	go tool golangci-lint run

lint-fix:
	go tool golangci-lint run --fix

build:
	go build

ci: fmt modernize-fix lint-fix test build cov-check

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
	brew install k1LoW/tap/octocov

e2e-setup:
	cd e2e/terraform && terraform init && terraform apply -auto-approve

e2e-run:
	./e2e/e2e-test.sh

e2e-clean:
	cd e2e/terraform && ./cleanup.sh && terraform destroy -auto-approve

e2e-full: e2e-setup e2e-run e2e-clean
