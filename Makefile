I = "⚪"
E = "🔴"
D = "🔵"

default:
	@echo "$(D) supported commands: [init, update, test]"

init:
	@echo "$(I) initialiazing..."
	@rm -rf go.mod go.sum ./vendor ./mocks
	@go mod init $$(pwd | awk -F'/' '{print $$NF}') || (echo "$(E) initialization error"; exit 1)

update:
	@echo "$(I) installing dependencies..."
	@go get ./... || (echo "$(E) 'go get' error"; exit 1)
	@echo "$(I) updating imports..."
	@go mod tidy || (echo "$(E) 'go mod tidy' error"; exit 1)
	@echo "$(I) vendoring..."
	@go mod vendor || (echo "$(E) 'go mod vendor' error"; exit 1)
	@echo "$(I) regenerating mocks package..."
	# @mockery --name=<interface-name>
	# @mockery --name=<interface-name> --dir=vendor/github.com/<org>/<proj>/

test:
	@echo "$(I) linting..."
	@golangci-lint run ./... || (echo "$(E) linter error"; exit 1)
	@echo "$(I) unit testing..."
	@go test -v $$(go list ./... | grep -v vendor | grep -v mocks) -race -coverprofile=coverage.txt -covermode=atomic
