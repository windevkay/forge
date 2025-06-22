#!/bin/bash

# Script to create releases for the genie package
# Usage: ./scripts/release-genie.sh <version>
# Example: ./scripts/release-genie.sh v1.1.0

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.1.0"
    exit 1
fi

VERSION=$1

# Validate version format
if ! [[ $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format vX.Y.Z (e.g., v1.0.0)"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "genie/go.mod" ]; then
    echo "Error: Must be run from the forge repository root"
    exit 1
fi

# Check if working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: Working directory is not clean. Please commit or stash changes."
    exit 1
fi

# Check if we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "Warning: You're not on the main branch (current: $CURRENT_BRANCH)"
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

TAG="genie/$VERSION"

# Check if tag already exists
if git tag -l | grep -q "^$TAG$"; then
    echo "Error: Tag $TAG already exists"
    exit 1
fi

echo "Creating release $TAG for genie package..."

# Run tests before creating release
echo "Running tests..."
cd genie
go test -v
go mod tidy
cd ..

# Create and push tag
echo "Creating tag $TAG..."
git tag "$TAG"
git push origin "$TAG"

echo "✅ Successfully created release $TAG"
echo "Users can now install with: go get github.com/windevkay/forge/genie@$VERSION"
