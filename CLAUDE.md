# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

**Core commands:**
- `make build` - Build for current platform
- `make test` - Run all tests
- `make run ARGS="..."` - Run with arguments (e.g., `make run ARGS="install github:pyhub-kr/pyhub-mcptools"`)
- `go test ./internal/[module] -v` - Run tests for specific module

**Cross-platform building:**
- `make build-all` - Build for all platforms (Windows, macOS, Linux)
- `make package` - Create release packages with proper naming

**Code quality:**
- `make fmt` - Format Go code
- `make lint` - Run golangci-lint (requires golangci-lint installed)
- `make test-coverage` - Generate test coverage report

## Architecture Overview

This is a cross-platform installer CLI tool built with Go and Cobra framework. The application follows a modular architecture with clear separation of concerns:

**Main Entry Point:** `cmd/pyhub-installer/main.go`
- Defines two primary commands: `download` and `install`
- `download`: Downloads files from URLs with optional verification/extraction
- `install`: Installs from GitHub releases with platform auto-detection

**Core Modules (internal/):**
- `download/` - Chunk-based parallel downloading with progress bars
- `extract/` - Archive extraction (ZIP, TAR, TAR.GZ) with security checks and flatten options
- `github/` - GitHub API integration for release fetching and asset selection
- `install/` - File installation with permission setting and PATH management
- `verify/` - Cryptographic verification (SHA256/SHA512 checksums)

**Key Features:**
- Cross-compilation support for 5+ platforms
- Automatic platform detection for GitHub releases
- Security features: path traversal prevention, signature verification
- Advanced extraction: auto-flatten single top-level directories, remove archives after extraction
- Output directory auto-creation

**Dependencies:**
- `github.com/spf13/cobra` - CLI framework
- `github.com/schollz/progressbar/v3` - Progress visualization
- Standard library for HTTP, crypto, archive handling

## Testing Strategy

Each module has comprehensive test coverage:
- Unit tests for all core functionality
- Integration tests with real archive creation/extraction
- Security tests for path traversal prevention
- Platform-specific tests for file permissions and executable detection

Run specific module tests: `go test ./internal/extract -v -run TestFlatten`

## Build Process

The project uses GitHub Actions for automated releases:
- Triggered on version tags (v*)
- Cross-compiles for all platforms in single Ubuntu runner
- Generates SHA256 checksums automatically
- Creates platform-specific archives (tar.gz for Unix, zip for Windows)

## Module Dependencies

```
main.go
├── download/ (chunk downloading, progress bars)
├── github/ (API client, release parsing, asset selection)
├── extract/ (archive handling, security, flatten options)
├── install/ (file operations, permissions, PATH)
└── verify/ (checksum validation, signature support)
```

Each module is independently testable and has minimal external dependencies beyond standard library.