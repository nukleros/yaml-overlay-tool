VERSION ?= unstable
build:
	go build  \
		-o ./yot \
		-ldflags="-s -w -X github.com/nukleros/yaml-overlay-tool/internal/commands.version=$(VERSION) -extldflags '-static'" \
		./cmd/yot/

lint:
	golangci-lint run