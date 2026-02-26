VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BINARY := claude-workspace

# Sources that invalidate the build: all .go files, go.mod/sum, and embedded template assets
GO_SOURCES := $(shell find . -name '*.go' -not -path './bin/*' -not -path './.git/*') \
              $(wildcard go.mod go.sum) \
              $(shell find ./_template -type f 2>/dev/null)

.PHONY: build install test clean build-all vet lint smoke-test smoke-test-keep smoke-test-fast smoke-test-docker smoke-test-docker-fast check doc dev-docker dev-vm deploy-docker deploy-vm shell-docker shell-vm destroy-docker destroy-vm dep ensure-go ensure-cue

# ---------- dependency targets ----------
dep: ensure-go ensure-cue

ensure-go:
	@bash scripts/install-deps.sh --go

ensure-cue:
	@bash scripts/install-deps.sh --cue

# ---------- build / test / lint ----------
build: build-all

install: build
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)

test: ensure-go
	go test ./...

vet: ensure-go
	go vet ./...

lint: ensure-cue
	bash scripts/lint-templates.sh

# Serve godoc locally using pkgsite
GOBIN := $(shell go env GOPATH)/bin
PKGSITE := $(GOBIN)/pkgsite

doc: ensure-go $(PKGSITE)
	@echo "Starting pkgsite at http://localhost:6060"
	@echo "View docs at http://localhost:6060/github.com/lamchakchan/claude-workspace"
	$(PKGSITE) -http=localhost:6060 -open .

$(PKGSITE): | ensure-go
	go install golang.org/x/pkgsite/cmd/pkgsite@latest

clean:
	rm -f $(BINARY)
	rm -rf bin/

# Phony alias — delegates to the stamp file for change detection
build-all: bin/.stamp

# Real file target: only rebuilds when Go sources or template assets change
# ensure-go is order-only (|) so it doesn't force rebuilds — only GO_SOURCES drive that
bin/.stamp: $(GO_SOURCES) | ensure-go
	@mkdir -p bin
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 .
	GOOS=linux  GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 .
	GOOS=linux  GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 .
	chmod +x bin/$(BINARY)-*
	@rm -f $(BINARY)
	@ln -s bin/$(BINARY)-$$(go env GOOS)-$$(go env GOARCH) $(BINARY)
	@touch $@
	@echo "Built binaries:"
	@ls -lh bin/
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

# Pre-push validation (vet + test + lint + build)
check: vet test lint build
	@echo "All checks passed."

# Persistent dev environment (create + provision)
dev-docker:
	bash scripts/dev-env.sh create --docker

dev-vm:
	bash scripts/dev-env.sh create --vm

# Fast deploy to existing dev environment (uses pre-built binary from build-all)
deploy-docker: build-all
	PREBUILT_BINARY=bin/$(BINARY)-linux-$$(go env GOARCH) bash scripts/dev-env.sh deploy --docker

deploy-vm: build-all
	PREBUILT_BINARY=bin/$(BINARY)-linux-$$(go env GOARCH) bash scripts/dev-env.sh deploy --vm

# Interactive shell into dev environment (creates env if needed, deploys latest binary)
shell-docker: dev-docker deploy-docker
	bash scripts/dev-env.sh shell --docker

shell-vm: dev-vm deploy-vm
	bash scripts/dev-env.sh shell --vm

# Tear down dev environment
destroy-docker:
	bash scripts/dev-env.sh destroy --docker

destroy-vm:
	bash scripts/dev-env.sh destroy --vm
