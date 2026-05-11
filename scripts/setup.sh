#!/bin/bash
# Quick setup script for EmmAI

set -e

echo "EmmAI Setup Script"
echo "=================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or higher."
    echo "Visit: https://golang.org/dl/"
    exit 1
fi

echo "✓ Go is installed: $(go version)"
echo ""

# Check for OPENAI_API_KEY
if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠ OPENAI_API_KEY environment variable is not set"
    echo ""
    read -p "Enter your OpenAI API key (or press Enter to skip): " api_key
    if [ -n "$api_key" ]; then
        export OPENAI_API_KEY="$api_key"
        echo "export OPENAI_API_KEY=\"$api_key\"" >> ~/.bashrc
        echo "✓ API key set and added to ~/.bashrc"
    else
        echo "⚠ Skipping API key setup. You'll need to set it later."
    fi
else
    echo "✓ OPENAI_API_KEY is set"
fi
echo ""

# Build the application
echo "Building EmmAI..."
make build
echo "✓ Build complete"
echo ""

# Create config directory
mkdir -p ~/.emmai/conversations
echo "✓ Created config directory at ~/.emmai"
echo ""

# Copy example config if it doesn't exist
if [ ! -f ~/.emmai/config.yaml ]; then
    echo "Creating example config file..."
    cp configs/config.example.yaml ~/.emmai/config.yaml
    echo "✓ Config file created at ~/.emmai/config.yaml"
    echo "  You can edit this file to customize settings"
else
    echo "✓ Config file already exists at ~/.emmai/config.yaml"
fi
echo ""

echo "Setup complete! 🎉"
echo ""
echo "To run EmmAI:"
echo "  ./bin/emmai"
echo ""
echo "To install system-wide:"
echo "  make install"
echo ""
echo "For help:"
echo "  make help"
echo ""
