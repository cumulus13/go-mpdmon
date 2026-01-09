.PHONY: build test clean release install uninstall

BINARY_NAME=mpdmon
VERSION=$(shell git describe --tags --always --dirty)
GOBUILD=CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)"

build:
	$(GOBUILD) -o $(BINARY_NAME) .

build-all:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o dist/linux-amd64/$(BINARY_NAME) .
	GOOS=linux GOARCH=386 $(GOBUILD) -o dist/linux-386/$(BINARY_NAME) .
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o dist/linux-arm64/$(BINARY_NAME) .
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) -o dist/linux-arm/$(BINARY_NAME) .
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o dist/windows-amd64/$(BINARY_NAME).exe .
	GOOS=windows GOARCH=386 $(GOBUILD) -o dist/windows-386/$(BINARY_NAME).exe .
	GOOS=windows GOARCH=arm64 $(GOBUILD) -o dist/windows-arm64/$(BINARY_NAME).exe .
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o dist/darwin-amd64/$(BINARY_NAME) .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o dist/darwin-arm64/$(BINARY_NAME) .

test:
	go test -v ./...

clean:
	go clean
	rm -rf dist/ $(BINARY_NAME) $(BINARY_NAME).exe

release:
	goreleaser release --rm-dist --snapshot

install:
	go install .

uninstall:
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

docker-build:
	docker build -t mpdmon:latest .

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
	@echo "Available targets:"
	@echo "  build      - Build for current platform"
	@echo "  build-all  - Build for all platforms"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  release    - Create release snapshot"
	@echo "  install    - Install to GOPATH/bin"
	@echo "  uninstall  - Remove from GOPATH/bin"
	@echo "  fmt        - Format code"
	@echo "  vet        - Vet code"
	@echo "  lint       - Lint code"