.PHONY: build test install clean run lint fmt help

BINARY_NAME=emmai
INSTALL_PATH=/usr/local/bin
GO=go
GOFLAGS=-v

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) ./cmd/emmai

## test: Run tests
test:
	@echo "Running tests..."
	@$(GO) test $(GOFLAGS) ./...

## install: Install the binary to system
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@sudo cp bin/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installation complete! Run '$(BINARY_NAME)' to start."

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@$(GO) clean

## run: Build and run the application
run: build
	@./bin/$(BINARY_NAME)

## lint: Run linters
lint:
	@echo "Running linters..."
	@$(GO) vet ./...
	@which staticcheck > /dev/null && staticcheck ./... || echo "staticcheck not installed (go install honnef.co/go/tools/cmd/staticcheck@latest)"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
