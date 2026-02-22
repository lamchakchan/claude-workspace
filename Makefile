VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BINARY := claude-workspace

.PHONY: build install test clean build-all vet smoke-test smoke-test-keep smoke-test-fast smoke-test-docker smoke-test-docker-fast

build:
	go build $(LDFLAGS) -o $(BINARY) .

install: build
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf bin/

build-all: clean
	@mkdir -p bin
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 .
	GOOS=linux  GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 .
	GOOS=linux  GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 .
	@echo "Built binaries:"
	@ls -lh bin/

smoke-test:
	bash scripts/smoke-test.sh

smoke-test-keep:
	bash scripts/smoke-test.sh --keep

smoke-test-fast:
	bash scripts/smoke-test.sh --skip-claude-cli

smoke-test-docker:
	bash scripts/smoke-test.sh --docker

smoke-test-docker-fast:
	bash scripts/smoke-test.sh --docker --skip-claude-cli
