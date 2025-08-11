#!/bin/bash
# Exit immediately if a command exits with a non-zero status.
set -e

# Change to the project root directory to ensure all paths are correct.
cd "$(dirname "$0")/.."

echo "Starting multi-platform build with CGO enabled..."

# Create build directory if not exists
mkdir -p build

# --- Build for Linux AMD64 ---
echo "Building for Linux AMD64..."
CC="zig cc -target x86_64-linux-gnu" CXX="zig c++ -target x86_64-linux-gnu" CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -o build/mybatis-plus-generator-linux-amd64 ./cmd/generator/main.go

# --- Build for Linux ARM64 ---
echo "Building for Linux ARM64..."
CC="zig cc -target aarch64-linux-gnu" CXX="zig c++ -target aarch64-linux-gnu" CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -v -o build/mybatis-plus-generator-linux-arm64 ./cmd/generator/main.go

# --- Build for Darwin AMD64 (on macOS, native toolchain is fine) ---
echo "Building for Darwin AMD64..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -v -o build/mybatis-plus-generator-darwin-amd64 ./cmd/generator/main.go

# --- Build for Darwin ARM64 (on macOS, native toolchain is fine) ---
echo "Building for Darwin ARM64..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -v -o build/mybatis-plus-generator-darwin-arm64 ./cmd/generator/main.go

# --- Build for Windows AMD64 ---
echo "Building for Windows AMD64..."
CC="zig cc -target x86_64-windows-gnu" CXX="zig c++ -target x86_64-windows-gnu" CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -v -o build/mybatis-plus-generator-windows-amd64.exe ./cmd/generator/main.go

echo "All builds completed successfully!"
echo "Binaries are located in the build directory."