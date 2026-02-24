VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BINARY := claude-workspace

.PHONY: build install test clean build-all vet smoke-test smoke-test-keep smoke-test-fast smoke-test-docker smoke-test-docker-fast check dev-docker dev-vm deploy-docker deploy-vm shell-docker shell-vm destroy-docker destroy-vm

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
	chmod +x bin/$(BINARY)-*
	@echo "Built binaries:"
	@ls -lh bin/
	@rm -f $(BINARY)
	@ln -s bin/$(BINARY)-$$(go env GOOS)-$$(go env GOARCH) $(BINARY)
	@echo "Symlinked $(BINARY) -> bin/$(BINARY)-$$(go env GOOS)-$$(go env GOARCH)"
	@echo "$$PATH" | tr ':' '\n' | grep -qx '$(CURDIR)/bin' || echo "\nTo add to your PATH:\n  export PATH=\"$(CURDIR)/bin:$$PATH\""

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

# Pre-push validation (vet + test + build)
check: vet test build
	@echo "All checks passed."

# Persistent dev environment (create + provision)
dev-docker:
	bash scripts/dev-env.sh create --docker

dev-vm:
	bash scripts/dev-env.sh create --vm

# Fast deploy to existing dev environment (cross-compile + copy binary)
deploy-docker:
	bash scripts/dev-env.sh deploy --docker

deploy-vm:
	bash scripts/dev-env.sh deploy --vm

# Interactive shell into dev environment
shell-docker:
	bash scripts/dev-env.sh shell --docker

shell-vm:
	bash scripts/dev-env.sh shell --vm

# Tear down dev environment
destroy-docker:
	bash scripts/dev-env.sh destroy --docker

destroy-vm:
	bash scripts/dev-env.sh destroy --vm
