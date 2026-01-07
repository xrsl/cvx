# Getting Started

## Installation

```bash
go install github.com/xrsl/cvx@latest
```

## Prerequisites

- **git** and [GitHub CLI](https://cli.github.com/) (`gh`) - authenticated
- **AI CLI** (one of):
  - [Claude CLI](https://github.com/anthropics/claude-code)
  - [Gemini CLI](https://github.com/google-gemini/gemini-cli)
- **LaTeX** (for PDF generation):
  - [BasicTeX](https://tug.org/mactex/morepackages.html) (recommended for Mac)
  - [MacTeX](https://tug.org/mactex/)
  - [TeX Live](https://tug.org/texlive/)
- **[uv](https://docs.astral.sh/uv/)** - required for Python agent mode

## Setup

Run the initialization wizard in your repository:

```bash
cvx init
```

The wizard will:

1. **Link your GitHub repo** - auto-detected from git remote
2. **Select CLI agent** - Claude or Gemini
3. **Set CV source path** - your CV data file
4. **Set letter source path** - your letter data file
5. **Set reference directory** - additional context for AI
6. **Create/link GitHub Project** - with job-tracking statuses

## Your First Application

### 1. Add a Job Posting

```bash
cvx add https://company.com/careers/software-engineer
```

This will:

1. Fetch and parse the job posting
2. Extract structured details using AI
3. Create a GitHub issue
4. Add to your project board

### 2. Analyze Match Quality

```bash
cvx advise 42
```

Get AI-powered analysis of how well your CV matches the position.

### 3. Build Tailored Documents

**Interactive Mode (default):**

```bash
cvx build 42
```

Uses Claude or Gemini CLI for real-time editing with tool use.

**Python Agent Mode:**

```bash
cvx build -m sonnet-4
```

Structured YAML output with Pydantic validation.

### 4. Approve and Submit

```bash
cvx approve 42
```

Commits, tags, pushes, and updates project status to "Applied".

### 5. View Later

```bash
cvx view 42
```

Opens the PDF from the git tag.

## Workflow Summary

```
cvx init                    # Initialize project
cvx add <url>               # Add job posting
cvx advise <issue>          # Analyze job-CV match
cvx build [issue]           # Build tailored CV/cover letter
cvx approve [issue]         # Commit, tag, push
cvx view <issue>            # View submitted documents
```

## Build Modes

### Interactive CLI Mode (Default)

Best for iterative refinement:

```bash
cvx build 42                # Start/resume session
cvx build -c "more ML focus" # Add feedback
```

**Benefits:**

- Direct file editing by AI
- Session persistence
- Interactive conversation
- Auto-detects claude or gemini CLI

### Python Agent Mode

Best for structured, repeatable output:

```bash
cvx build -m sonnet-4       # Claude
cvx build -m flash-2-5      # Gemini
cvx build -m qwen3-32b      # Groq
```

**Benefits:**

- Schema-validated output
- Multi-provider support
- Automatic fallback on failure
- TOML output

**Requirements:**

- [uv](https://docs.astral.sh/uv/) installed
- Data files: `src/cv.toml`, `src/letter.toml`
- Schema: `schema/schema.json`

## Other Commands

```bash
cvx list                    # View all applications
cvx list --company google   # Filter by company
cvx rm <issue>              # Remove application
```

## Next Steps

- [Commands](commands.md) - Full command reference
- [Configuration](configuration.md) - Customize settings
- [Architecture](architecture.md) - How it works
- [Schema Reference](schema.md) - YAML/TOML structure
