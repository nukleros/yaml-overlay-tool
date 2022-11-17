#
# tests
#
GOLANGCI_LINT_VERSION ?= v1.50.1
install-linter:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

test-unit:
	@go test -cover -coverprofile=./coverage.out ./...

lint-go:
	golangci-lint run

lint-yaml:
	@yamllint ./
