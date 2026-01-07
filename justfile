
# Default recipe (shows help)
default:
    @just --list

# print out tree
tree:
    tree -a -L 4 -I ".venv|__pycache__|.git"

# Run all tests (Go + Python)
test: test-go test-python
    @echo "✅ All tests passed!"

# Run Go tests
test-go:
    @echo "🧪 Running Go tests..."
    go test ./cmd/... -v

# Run Python tests (unit + integration)
test-python:
    @echo "🐍 Running Python tests..."
    cd agent && uv run pytest -v

# Run tests with coverage
test-coverage:
    @echo "📊 Running tests with coverage..."
    go test ./cmd/... -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    cd agent && pytest --cov=cvx_agent --cov-report=html
    @echo "✅ Coverage reports generated:"
    @echo "  - Go: coverage.html"
    @echo "  - Python: agent/htmlcov/index.html"

# Build and install
build:
    make

# Clean build artifacts
clean:
    rm -rf coverage.out coverage.html
    rm -rf agent/htmlcov agent/.coverage
    rm -rf agent/.pytest_cache
    rm -f agent/test_input.json agent/output.json

serve-docs:
    @echo "🚀 Serving docs..."
    uv run mkdocs serve


prek:
    prek run --all-files
