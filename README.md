# pyhub-installer

A fast, secure cross-platform installer for downloading, verifying and installing files from URLs or GitHub releases.

**Standalone project** - This is an independent CLI tool that can be used with any project, not just pyhub-mcptools.

## Features

- **Fast parallel downloads** with chunk-based downloading
- **Automatic signature verification** (SHA256, auto-detect from GitHub releases)
- **Archive extraction** (ZIP, TAR, TAR.GZ, GZIP)
- **Cross-platform support** (Windows, macOS, Linux)
- **GitHub release integration** with automatic platform detection
- **Executable permissions** automatically set on Unix systems

## Installation

### Download from GitHub Releases

```bash
# macOS (Intel)
curl -L https://github.com/pyhub-kr/pyhub-installer/releases/latest/download/pyhub-installer-darwin-amd64.tar.gz | tar -xz

# macOS (Apple Silicon)
curl -L https://github.com/pyhub-kr/pyhub-installer/releases/latest/download/pyhub-installer-darwin-arm64.tar.gz | tar -xz

# Linux (x64)
curl -L https://github.com/pyhub-kr/pyhub-installer/releases/latest/download/pyhub-installer-linux-amd64.tar.gz | tar -xz

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/pyhub-kr/pyhub-installer/releases/latest/download/pyhub-installer-windows-amd64.zip" -OutFile "pyhub-installer.zip"
Expand-Archive pyhub-installer.zip -DestinationPath .
```

### Build from Source

```bash
git clone https://github.com/pyhub-kr/pyhub-installer.git
cd installer
make build
```

## Usage

### Download and Install Files

```bash
# Basic download
pyhub-installer download https://example.com/file.zip

# Download with verification
pyhub-installer download https://example.com/file.zip \
  --verify --signature https://example.com/file.zip.sha256

# Download and extract
pyhub-installer download https://example.com/archive.zip \
  --extract --output /usr/local/bin

# Set custom permissions
pyhub-installer download https://example.com/binary \
  --chmod 755 --output /usr/local/bin
```

### Install from GitHub Releases

```bash
# Install latest release (auto-detect platform)
pyhub-installer install github:cli/cli

# Install specific version
pyhub-installer install github:cli/cli --version v2.40.0

# Install to custom directory
pyhub-installer install github:cli/cli --output ./tools

# Install for specific platform
pyhub-installer install github:cli/cli --platform linux-amd64
```

### Command Options

#### Download Command
- `--output, -o`: Output directory (default: current directory)
- `--verify, -v`: Verify file signature
- `--extract, -x`: Extract archive after download
- `--signature, -s`: URL of signature file for verification
- `--chmod`: Set file permissions (Unix only, default: 755)

#### Install Command
- `--version`: Version to install (default: latest)
- `--platform`: Target platform (auto-detect if not specified)
- `--output, -o`: Installation directory (default: /usr/local/bin)

## Examples

### Install GitHub CLI
```bash
pyhub-installer install github:cli/cli
```

### Install Specific Tools
```bash
# Install Hugo static site generator
pyhub-installer install github:gohugoio/hugo

# Install kubectl
pyhub-installer install github:kubernetes/kubernetes --platform linux-amd64

# Install to custom location
pyhub-installer install github:golang/go --output ./go --version v1.21.0
```

### Download and Verify Files
```bash
# Download with automatic signature verification
pyhub-installer download https://releases.example.com/tool.tar.gz \
  --verify --signature https://releases.example.com/tool.tar.gz.sha256sum

# Download and extract to /usr/local/bin
pyhub-installer download https://example.com/tool.zip \
  --extract --output /usr/local/bin --chmod 755
```

## Performance

- **5-10x faster** than PowerShell/curl for large files
- **Parallel chunk downloads** with automatic optimization
- **Progress bars** for visual feedback
- **Automatic retry** on network failures

## Platform Support

| Platform | Architecture | Status |
|----------|-------------|---------|
| Linux | x86_64 | ✅ |
| Linux | ARM64 | ✅ |
| Linux | x86 | ✅ |
| macOS | Intel | ✅ |
| macOS | Apple Silicon | ✅ |
| Windows | x64 | ✅ |
| Windows | x86 | ✅ |

## Verification Support

- **SHA256** checksums
- **SHA512** checksums (planned)
- **GPG signatures** (planned)
- **Automatic detection** from GitHub releases

## Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Create release packages
make package

# Run tests
make test

# Install locally
make install
```

## Development

Requirements:
- Go 1.21 or later
- Make

```bash
# Install dependencies
make deps

# Format code
make fmt

# Run linter
make lint

# Run with arguments
make run ARGS="install github:cli/cli"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make test`
6. Submit a pull request

## License

This project is an independent CLI tool licensed under MIT License.