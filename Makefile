.PHONY: all build install clean docs docs-serve

# Default target
all: build install clean

build:
	@go build -o cvx .

install: build
	@mkdir -p ~/.local/bin
	@cp cvx ~/.local/bin/
	@echo "Installed to ~/.local/bin/cvx"

clean:
	@rm -f cvx

# Documentation
docs:
	@uv run mkdocs build

docs-serve:
	@uv run mkdocs serve
