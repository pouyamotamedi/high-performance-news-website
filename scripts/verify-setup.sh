#!/bin/bash

# Verification script for High Performance News Website setup

echo "🔍 Verifying High Performance News Website setup..."
echo

# Check Go installation
echo "Checking Go installation..."
if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    echo "✅ Go is installed: $GO_VERSION"
    
    # Check Go version (requires 1.21+)
    GO_VERSION_NUM=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    MAJOR=$(echo $GO_VERSION_NUM | cut -d. -f1)
    MINOR=$(echo $GO_VERSION_NUM | cut -d. -f2)
    
    if [ "$MAJOR" -gt 1 ] || ([ "$MAJOR" -eq 1 ] && [ "$MINOR" -ge 21 ]); then
        echo "✅ Go version is compatible (requires 1.21+)"
    else
        echo "❌ Go version $GO_VERSION_NUM is too old (requires 1.21+)"
        echo "   Please upgrade Go: https://golang.org/dl/"
    fi
else
    echo "❌ Go is not installed"
    echo "   Please install Go: https://golang.org/dl/"
fi
echo

# Check Docker installation
echo "Checking Docker installation..."
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version)
    echo "✅ Docker is installed: $DOCKER_VERSION"
else
    echo "❌ Docker is not installed"
    echo "   Please install Docker: https://docs.docker.com/get-docker/"
fi
echo

# Check Docker Compose installation
echo "Checking Docker Compose installation..."
if command -v docker-compose &> /dev/null; then
    COMPOSE_VERSION=$(docker-compose --version)
    echo "✅ Docker Compose is installed: $COMPOSE_VERSION"
elif docker compose version &> /dev/null; then
    COMPOSE_VERSION=$(docker compose version)
    echo "✅ Docker Compose (plugin) is installed: $COMPOSE_VERSION"
else
    echo "❌ Docker Compose is not installed"
    echo "   Please install Docker Compose: https://docs.docker.com/compose/install/"
fi
echo

# Check Make installation
echo "Checking Make installation..."
if command -v make &> /dev/null; then
    MAKE_VERSION=$(make --version | head -n1)
    echo "✅ Make is installed: $MAKE_VERSION"
else
    echo "⚠️  Make is not installed (optional but recommended)"
    echo "   On Ubuntu/Debian: sudo apt-get install build-essential"
    echo "   On macOS: xcode-select --install"
    echo "   On Windows: Install via chocolatey or use WSL"
fi
echo

# Check project structure
echo "Checking project structure..."
REQUIRED_DIRS=("cmd" "internal" "pkg" "web" "migrations" "configs")
REQUIRED_FILES=("go.mod" "docker-compose.yml" "Makefile" ".air.toml")

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo "✅ Directory exists: $dir"
    else
        echo "❌ Missing directory: $dir"
    fi
done

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ File exists: $file"
    else
        echo "❌ Missing file: $file"
    fi
done
echo

# Check Go module
echo "Checking Go module..."
if [ -f "go.mod" ]; then
    if command -v go &> /dev/null; then
        echo "Running go mod tidy..."
        if go mod tidy; then
            echo "✅ Go module is valid"
        else
            echo "❌ Go module has issues"
        fi
    else
        echo "⚠️  Cannot verify Go module (Go not installed)"
    fi
else
    echo "❌ go.mod file not found"
fi
echo

echo "🎉 Setup verification complete!"
echo
echo "Next steps:"
echo "1. Install any missing dependencies listed above"
echo "2. Copy .env.example to .env and configure as needed"
echo "3. Run 'make dev-setup' to complete development environment setup"
echo "4. Run 'make dev' to start the development server"