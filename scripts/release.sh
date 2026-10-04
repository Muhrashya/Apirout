#!/usr/bin/env bash
set -euo pipefail

# Automated Release Script for POSIX environments

VERSION="${1:-}"
PUSH="${2:-}"

if [ -z "$VERSION" ]; then
    echo "Usage: ./scripts/release.sh <version> [--push]"
    echo "Example: ./scripts/release.sh v1.0.0 --push"
    exit 1
fi

TAG="${VERSION}"
if [[ ! "$TAG" =~ ^v ]]; then
    TAG="v${TAG}"
fi
SEMVER="${TAG#v}"

echo "==> Starting release preparation for $TAG ($SEMVER)..."

# Check if tag already exists
if git rev-parse "$TAG" >/dev/null 2>&1; then
    echo "Error: Tag '$TAG' already exists!"
    exit 1
fi

# Check for clean working directory
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: Working directory has uncommitted changes. Please commit or stash changes before releasing."
    exit 1
fi

# Run test suite
echo "==> Running full test suite..."
go test -count=1 ./...

# Run web build
echo "==> Verifying web console build..."
(cd web && npm run build)

# Sync package.json version
if [ -f "web/package.json" ]; then
    if grep -q "\"version\":" web/package.json; then
        sed -i.bak -E "s/\"version\": \"[^\"]+\"/\"version\": \"$SEMVER\"/" web/package.json
        rm -f web/package.json.bak
        if [ -n "$(git status --porcelain web/package.json)" ]; then
            git add web/package.json
            git commit -m "chore(release): bump package manifest to $TAG"
        fi
    fi
fi

# Create annotated tag
echo "==> Creating annotated tag $TAG..."
git tag -a "$TAG" -m "Release $TAG - Apirout Open Source API Engine & Gateway"
echo "==> Tag $TAG created successfully!"

if [ "$PUSH" == "--push" ]; then
    echo "==> Pushing commits and tag $TAG to origin..."
    git push origin main --tags
    echo "==> Successfully released and pushed $TAG to origin!"
else
    echo ""
    echo "Release $TAG prepared locally. To push to GitHub, run:"
    echo "  git push origin main --tags"
fi
