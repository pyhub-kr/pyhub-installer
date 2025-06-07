# Makefile for pyhub-installer

# Variables
BINARY_NAME=pyhub-installer
MAIN_PATH=./cmd/pyhub-installer
BUILD_DIR=./build
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	@echo "Building ${BINARY_NAME} for current platform..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${MAIN_PATH}

# Build for all platforms
.PHONY: build-all
build-all: build-windows build-macos build-linux

# Build for Windows
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p ${BUILD_DIR}/windows-amd64
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/windows-amd64/${BINARY_NAME}.exe ${MAIN_PATH}
	@mkdir -p ${BUILD_DIR}/windows-386
	GOOS=windows GOARCH=386 go build ${LDFLAGS} -o ${BUILD_DIR}/windows-386/${BINARY_NAME}.exe ${MAIN_PATH}

# Build for macOS
.PHONY: build-macos
build-macos:
	@echo "Building for macOS..."
	@mkdir -p ${BUILD_DIR}/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/darwin-amd64/${BINARY_NAME} ${MAIN_PATH}
	@mkdir -p ${BUILD_DIR}/darwin-arm64
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/darwin-arm64/${BINARY_NAME} ${MAIN_PATH}

# Build for Linux
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p ${BUILD_DIR}/linux-amd64
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/linux-amd64/${BINARY_NAME} ${MAIN_PATH}
	@mkdir -p ${BUILD_DIR}/linux-386
	GOOS=linux GOARCH=386 go build ${LDFLAGS} -o ${BUILD_DIR}/linux-386/${BINARY_NAME} ${MAIN_PATH}
	@mkdir -p ${BUILD_DIR}/linux-arm64
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/linux-arm64/${BINARY_NAME} ${MAIN_PATH}

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf ${BUILD_DIR}
	rm -f coverage.out coverage.html

# Install to local system
.PHONY: install
install: build
	@echo "Installing ${BINARY_NAME} to /usr/local/bin..."
	sudo cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/

# Create release packages
.PHONY: package
package: build-all
	@echo "Creating release packages..."
	cd ${BUILD_DIR} && \
	for dir in */; do \
		platform=$${dir%/}; \
		echo "Packaging $${platform}..."; \
		if [[ "$${platform}" == *"windows"* ]]; then \
			zip -r ${BINARY_NAME}-${VERSION}-$${platform}.zip $${platform}/; \
		else \
			tar -czf ${BINARY_NAME}-${VERSION}-$${platform}.tar.gz $${platform}/; \
		fi; \
	done

# Development run
.PHONY: run
run:
	go run ${MAIN_PATH} $(ARGS)

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build for current platform"
	@echo "  build-all    - Build for all platforms"
	@echo "  build-windows- Build for Windows"
	@echo "  build-macos  - Build for macOS"
	@echo "  build-linux  - Build for Linux"
	@echo "  deps         - Install dependencies"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install to local system"
	@echo "  package      - Create release packages"
	@echo "  run          - Run development version"
	@echo "  help         - Show this help"