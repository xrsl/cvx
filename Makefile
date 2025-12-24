.PHONY: all build install clean docs docs-serve cv letter app distclean help

# Configuration
LATEX = xelatex
LATEXMK = latexmk
BUILD_DIR = build
SRC_DIR = src

# PDF targets
CV_SRC = $(SRC_DIR)/cv.tex
CV_PDF = $(BUILD_DIR)/cv.pdf
LETTER_SRC = $(SRC_DIR)/letter.tex
LETTER_PDF = $(BUILD_DIR)/letter.pdf
COMBINED_PDF = $(BUILD_DIR)/combined.pdf
STYLE = $(SRC_DIR)/cv-style.sty

# LaTeX flags
LATEXMK_FLAGS = -pdf -f -output-directory=$(BUILD_DIR) -interaction=nonstopmode

# Default target - build CLI
all: build install clean

# Build CLI
build:
	@go build -o cvx .

# Install CLI
install: build
	@mkdir -p ~/.local/bin
	@cp cvx ~/.local/bin/
	@echo "Installed to ~/.local/bin/cvx"

# Clean CLI binary
clean:
	@rm -f cvx

# Build CV PDF
cv:
	@echo "Building CV..."
	@mkdir -p $(BUILD_DIR)
	@cp $(STYLE) $(BUILD_DIR)/
	@$(LATEXMK) $(LATEXMK_FLAGS) $(CV_SRC) > /dev/null 2>&1
	@echo "✓ CV built: $(CV_PDF)"

# Build cover letter PDF
letter:
	@echo "Building cover letter..."
	@mkdir -p $(BUILD_DIR)
	@$(LATEXMK) $(LATEXMK_FLAGS) $(LETTER_SRC) > /dev/null 2>&1
	@echo "✓ Letter built: $(LETTER_PDF)"

# Combine CV and letter into application PDF
app: cv letter
	@echo "Building application..."
	@python3 -c "from pypdf import PdfWriter; w = PdfWriter(); [w.append(f) for f in ['$(CV_PDF)', '$(LETTER_PDF)']]; w.write('$(COMBINED_PDF)')" 2>/dev/null || \
	(echo "Installing pypdf..." && python3 -m pip install -q pypdf && \
	python3 -c "from pypdf import PdfWriter; w = PdfWriter(); [w.append(f) for f in ['$(CV_PDF)', '$(LETTER_PDF)']]; w.write('$(COMBINED_PDF)')")
	@echo "✓ Application PDF: $(COMBINED_PDF)"

# Deep clean (remove build directory)
distclean:
	@echo "Removing build directory..."
	@rm -rf $(BUILD_DIR)
	@rm -f cvx
	@echo "✓ Clean complete"

# Documentation
docs:
	@uv run mkdocs build

docs-serve:
	@uv run mkdocs serve

# Help
help:
	@echo "cvx Build System"
	@echo "================"
	@echo ""
	@echo "CLI:"
	@echo "  make          - Build and install CLI (default)"
	@echo "  make build    - Build CLI binary"
	@echo "  make install  - Install CLI to ~/.local/bin"
	@echo ""
	@echo "Documents:"
	@echo "  make cv       - Build CV PDF"
	@echo "  make letter   - Build cover letter PDF"
	@echo "  make app      - Build application PDF (CV + letter)"
	@echo ""
	@echo "Other:"
	@echo "  make docs     - Build documentation"
	@echo "  make clean    - Remove CLI binary"
	@echo "  make distclean- Remove all build artifacts"
	@echo "  make help     - Show this help"
