# cvx

**AI-powered CLI for CV tailoring and job application tracking.**

`cvx` uses AI to extract job details from any job posting URL, tracks your applications in GitHub Issues + Projects, and helps you tailor your CV and cover letter — all from your terminal.

## What it does

- **Extracts job details** from URLs using AI (Claude or Gemini)
- **Creates GitHub Issues** with structured job information
- **Tracks applications** in a GitHub Project with status, company, and deadlines
- **Analyzes job-CV match** quality with AI-powered career advice
- **Tailors CV and cover letter** using:
  - **Python Agent Mode**: Structured YAML/TOML output with Pydantic validation
  - **Interactive CLI Mode**: Direct file editing with session persistence

## Quick Example

```bash
# Initialize in your repo
cvx init

# Add a job posting
cvx add https://company.com/careers/software-engineer

# Analyze match quality
cvx advise 42

# Build tailored CV/cover letter (Python agent mode)
cvx build -m sonnet-4

# Or use interactive CLI mode (default)
cvx build 42

# Approve and finalize
cvx approve 42

# View submitted documents
cvx view 42
```

## Branching and Tagging Strategy

`cvx` uses git branches for building and git tags for archiving submitted applications:

```bash
# Build CV/letter (auto-creates branch 42-company-role)
cvx build 42

# Once satisfied, approve to commit, tag, and push
cvx approve 42

# This creates git tag in the following format:
# 42-company-role-2025-12-18

# Later, view what you submitted
cvx view 42
```

**Branch format**: `{issue}-{company}-{role}`

**Tag format**: `{issue}-{company}-{role}-{application-date}`

This keeps a permanent record of exactly what you sent to each company.

## Requirements

- `git` and [GitHub CLI](https://cli.github.com/) (`gh`) - installed and authenticated
- One of: [Claude CLI](https://github.com/anthropics/claude-code), [Gemini CLI](https://github.com/google-gemini/gemini-cli), or API keys
- LaTeX: [BasicTeX](https://tug.org/mactex/morepackages.html), [MacTeX](https://tug.org/mactex/), or [TeX Live](https://tug.org/texlive/)
- [uv](https://docs.astral.sh/uv/) - required for Python agent mode

## Documentation

- [Getting Started](getting-started.md) - Installation and first steps
- [Commands](commands.md) - Detailed command reference
- [Configuration](configuration.md) - Config file and customization
- [Architecture](architecture.md) - How cvx works internally
- [Schema Reference](schema.md) - YAML/TOML structure for Python agent mode
