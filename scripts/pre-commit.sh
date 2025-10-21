#!/bin/bash
# Pre-commit checks for go-gedcom
# Run this before committing to catch issues that CI will catch

set -e

echo "🔍 Running pre-commit checks..."
echo ""

# Change to project root
cd "$(git rev-parse --show-toplevel)"

# 1. Check formatting
echo "1️⃣  Checking Go formatting..."
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
  echo "❌ Error: The following files are not formatted:"
  echo "$UNFORMATTED"
  echo ""
  echo "Run: gofmt -w ."
  exit 1
fi
echo "✅ All files are properly formatted"
echo ""

# 2. Run go vet
echo "2️⃣  Running go vet..."
if ! go vet ./...; then
  echo "❌ go vet failed"
  exit 1
fi
echo "✅ go vet passed"
echo ""

# 3. Run tests
echo "3️⃣  Running tests..."
if ! go test ./... -short; then
  echo "❌ Tests failed"
  exit 1
fi
echo "✅ Tests passed"
echo ""

# 4. Check coverage
echo "4️⃣  Checking test coverage..."
COVERAGE=$(go test -cover ./charset ./decoder ./encoder ./gedcom ./parser ./validator ./version 2>&1 | grep -oE '[0-9]+\.[0-9]+%' | tail -1 | sed 's/%//')
if [ -z "$COVERAGE" ]; then
  COVERAGE="0.0"
fi

echo "📊 Total coverage: ${COVERAGE}%"
if (( $(echo "$COVERAGE < 85.0" | bc -l) )); then
  echo "❌ Error: Test coverage ($COVERAGE%) is below 85%"
  exit 1
fi
echo "✅ Coverage requirement met (≥85%)"
echo ""

# 5. Run linter (if available)
if command -v golangci-lint &> /dev/null; then
  echo "5️⃣  Running golangci-lint..."
  if ! golangci-lint run --timeout=5m ./...; then
    echo "❌ Linting failed"
    exit 1
  fi
  echo "✅ Linting passed"
else
  echo "⚠️  golangci-lint not found (skipping)"
  echo "   Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi
echo ""

echo "✅ All pre-commit checks passed!"
echo ""
echo "💡 Tip: To install as a git hook, run:"
echo "   ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit"
