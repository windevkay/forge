#!/bin/bash

# Script to run golangci-lint locally with the local configuration
# This uses .golangci-local.yml which is compatible with golangci-lint v2.x

set -e

echo "Running golangci-lint locally with .golangci-local.yml config..."

# Check if .golangci-local.yml exists
if [ ! -f ".golangci-local.yml" ]; then
    echo "Error: .golangci-local.yml not found in current directory"
    exit 1
fi

# Run linting for genie package
echo "Linting genie package..."
cd genie
golangci-lint run --config ../.golangci-local.yml
cd ..

# Run linting for flho package
echo "Linting flho package..."
cd flho
golangci-lint run --config ../.golangci-local.yml
cd ..

echo "Local linting completed successfully!"
