.PHONY: build test clean release install uninstall

BINARY_NAME=mpdmon
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD)")
GOBUILD=CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)"
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

build:
	$(GOBUILD) -o $(BINARY_NAME) .

build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME).exe .

build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-linux .

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-macos .

build-all:
	make build-windows
	make build-linux
	make build-darwin

test:
	go test -v ./...

clean:
	go clean
	rm -rf dist/ $(BINARY_NAME)*

release:
ifeq ($(GIT_BRANCH),master)
	@echo "On master branch, running goreleaser..."
	goreleaser release --rm-dist
else
	@echo "Not on master branch, creating snapshot..."
	goreleaser release --rm-dist --snapshot
endif

install:
	go install .

uninstall:
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

docker-build:
	docker build -t mpdmon:$(VERSION) .

docker-run:
	docker run --rm -it --network=host mpdmon:latest

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

.PHONY: help
help:
	@echo "MPD Monitor Build System"
	@echo ""
	@echo "Current branch: $(GIT_BRANCH)"
	@echo "Version: $(VERSION)"
	@echo ""
	@echo "Available targets:"
	@echo "  build           - Build for current platform"
	@echo "  build-windows   - Build for Windows"
	@echo "  build-linux     - Build for Linux"
	@echo "  build-darwin    - Build for macOS"
	@echo "  build-all       - Build for all major platforms"
	@echo "  test            - Run tests"
	@echo "  clean           - Clean build artifacts"
	@echo "  release         - Create release (master only) or snapshot"
	@echo "  install         - Install to GOPATH/bin"
	@echo "  uninstall       - Remove from GOPATH/bin"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run Docker container"
	@echo "  fmt             - Format code"
	@echo "  vet             - Vet code"
	@echo "  lint            - Lint code"