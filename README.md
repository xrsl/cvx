[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![CI](https://github.com/xrsl/cvx/actions/workflows/ci.yml/badge.svg)](https://github.com/xrsl/cvx/actions/workflows/ci.yml)
[![docs](https://github.com/xrsl/cvx/actions/workflows/docs.yml/badge.svg)](https://github.com/xrsl/cvx/actions/workflows/docs.yml)
[![codecov](https://codecov.io/gh/xrsl/cvx/graph/badge.svg)](https://codecov.io/gh/xrsl/cvx)
[![prek](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/j178/prek/master/docs/assets/badge-v0.json)](https://github.com/j178/prek)
[![GitHub release](https://img.shields.io/github/v/release/xrsl/cvx?style=flat&color=blue)](https://github.com/xrsl/cvx/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Claude](https://img.shields.io/badge/Claude-Powered-cc785c?style=flat&logo=anthropic&logoColor=white)](https://claude.ai)
[![Gemini](https://img.shields.io/badge/Gemini-Powered-4285F4?style=flat&logo=google&logoColor=white)](https://gemini.google.com)

# cvx

`cvx` uses AI to extract job details from any job posting URL, tracks your applications in GitHub Issues + Projects, and helps you tailor your CV and cover letter using LaTeX — all from your terminal.

## What it does

- **Extracts job details** from URLs using AI agents (Claude Code or Gemini CLI)
- **Creates GitHub Issues** with structured job information using a custimizable template (default `job-ad-schema.yaml`).
- **Tracks applications in a GitHub Project** with status, company, and deadlines
- **Analyzes job-CV match** quality with AI agents
- **Tailors CV and cover letter** with AI agents by editing your LaTeX source files

## Installation

```bash
go install github.com/xrsl/cvx@latest
```

## Requirements

- `git` and [GitHub CLI](https://cli.github.com/) (`gh`) - installed and authenticated
- One of: [Claude Code CLI](https://github.com/anthropics/claude-code), [Gemini CLI](https://github.com/google-gemini/gemini-cli), or an API key (`ANTHROPIC_API_KEY` or `GEMINI_API_KEY`)
- LaTeX: [BasicTeX](https://tug.org/mactex/morepackages.html) (light, recommended for Mac), [MacTeX](https://tug.org/mactex/), or [TeX Live](https://tug.org/texlive/) - for building PDFs
- [uv](https://docs.astral.sh/uv/) - required for Python agent mode (`cvx build -m`)

## Quickstart

```bash
cvx init                              # Setup wizard
cvx add https://company.com/job       # Add job posting
cvx advise 42                         # Analyze job-CV match
cvx build 42                          # Build tailored CV/cover letter
cvx approve 42                        # Commit, tag, push, update status
cvx view 42                           # View submitted documents
```

## Commands

### `cvx add <url>`

Fetches job posting, extracts details with AI, creates GitHub issue.

```bash
cvx add https://company.com/job
cvx add https://company.com/job --dry-run          # extract only
cvx add https://company.com/job -a gemini          # use Gemini CLI
cvx add https://company.com/job -m claude-sonnet-4 # use Claude API
cvx add https://company.com/job --body             # use .cvx/body.md
cvx add https://company.com/job -b job.md          # use custom file
```

### `cvx list`

Lists all job applications with status, company, and deadline.

```bash
cvx list
cvx list --state closed   # show closed issues
cvx list --company google # filter by company
```

### `cvx advise <issue>`

Get career advice on job match quality.

```bash
cvx advise 42                         # Analyze issue #42
cvx advise 42 --push                  # Post analysis as comment
cvx advise 42 -a gemini               # Use Gemini CLI
cvx advise 42 -m gemini-2.5-flash     # Use Gemini API
cvx advise 42 -i                      # Interactive session
cvx advise https://example.com/job
```

### `cvx build [issue]`

Build tailored CV and cover letter. Automatically creates/switches to the issue branch (`42-company-role`).

**Two Build Modes:**

1. **Python Agent Mode** (default when using `-m`): Uses structured YAML output with schema validation
2. **CLI Agent Mode**: Uses Claude Code or Gemini CLI for interactive/non-interactive sessions

```bash
# Python Agent Mode (structured YAML)
cvx build -m claude-sonnet-4         # Use Python agent with Claude
cvx build -m gemini-2.5-flash        # Use Python agent with Gemini
cvx build -m sonnet-4 --dry-run      # Preview without calling AI
cvx build -m sonnet-4 --no-cache     # Skip cache

# CLI Agent Mode (LaTeX editing)
cvx build                            # Infer issue from branch
cvx build 42                         # Build for issue #42
cvx build -i                         # Interactive session
cvx build -a claude                  # Use Claude CLI
cvx build -a gemini                  # Use Gemini CLI
cvx build -c "emphasize Python"      # Continue with feedback

# Common options
cvx build -o                         # Build and open PDF
cvx build --commit --push            # Build, commit, and push

# Direct API mode (legacy)
cvx build -m sonnet-4 --call-api-directly
```

**Python Agent Mode** requires [uv](https://docs.astral.sh/uv/) and works with YAML files (`src/cv.yaml`, `src/letter.yaml`) validated against `schema/schema.json`. It provides structured output, automatic caching, and multi-provider fallback.

Sessions are shared per issue. Use `-c` to continue with feedback.

### `cvx approve [issue]`

Approve and finalize the tailored application: commits, tags, pushes, updates project status.

```bash
cvx approve                          # Infer issue from branch
cvx approve 42                       # Approve issue #42
```

### `cvx view <issue>`

View submitted application documents.

```bash
cvx view 42                      # Open combined or CV PDF
cvx view 42 -l                   # Open cover letter
cvx view 42 -c                   # Open CV only
```

Opens the PDF from the git tag. Tag format: `{issue}-{company}-{role}-{date}` (e.g., `42-saxo-bank-senior-data-scientist-2025-12-18`)

### `cvx rm <issue>`

Deletes an issue.

```bash
cvx rm 42
```

### `cvx init`

Interactive setup wizard.

```bash
cvx init                # interactive wizard
cvx init -q             # quiet mode with defaults
cvx init -r             # reset workflows to defaults
cvx init -c             # validate config resources
cvx init -d             # delete .cvx/ and config
```

## AI Agents and Models

Use `--agent/-a` for CLI tools or `--model/-m` for API access:

```bash
cvx add https://job.com -a claude            # Claude CLI
cvx add https://job.com -m claude-sonnet-4   # Claude API
cvx advise 42 -a gemini                      # Gemini CLI
cvx advise 42 -m gemini-2.5-flash            # Gemini API
```

### CLI Agents (`--agent`)

| Agent    | Notes                                                        |
| -------- | ------------------------------------------------------------ |
| `claude` | [Claude Code CLI](https://github.com/anthropics/claude-code) |
| `gemini` | [Gemini CLI](https://github.com/google-gemini/gemini-cli)    |

### API Models (`--model`)

| Model                    | Notes                        |
| ------------------------ | ---------------------------- |
| `claude-sonnet-4`        | Requires `ANTHROPIC_API_KEY` |
| `claude-sonnet-4-5`      | Requires `ANTHROPIC_API_KEY` |
| `claude-opus-4`          | Requires `ANTHROPIC_API_KEY` |
| `claude-opus-4-5`        | Requires `ANTHROPIC_API_KEY` |
| `gemini-2.5-flash`       | Requires `GEMINI_API_KEY`    |
| `gemini-2.5-pro`         | Requires `GEMINI_API_KEY`    |
| `gemini-3-flash-preview` | Requires `GEMINI_API_KEY`    |
| `gemini-3-pro-preview`   | Requires `GEMINI_API_KEY`    |

Priority order for default: CLI agents first (claude-code > gemini-cli), then API models.

## GitHub Project

cvx automatically creates a GitHub Project with:

**Fields:**

- Application Status (single-select)
- Company (text)
- Deadline (date)
- AppliedDate (date)

**Statuses:**

- To be Applied
- Applied
- Interview
- Offered
- Accepted
- Gone
- Let Go

Issues are automatically added to the project when created with `cvx add`.

## Config File

Located at `.cvx-config.yaml` in your repo root:

```yaml
repo: owner/repo
agent: claude-code
cv_path: src/cv.tex
reference_path: reference/
project: owner/1
```

The `reference_path` directory should contain your experience documentation, guidelines, and other reference materials used by `advise` and `build` commands.

Internal project IDs are cached in `.cvx/cache.yaml` (auto-managed).

## Customizing Workflows

AI prompts are stored in `.cvx/workflows/` and can be customized:

| File        | Used by                              |
| ----------- | ------------------------------------ |
| `add.md`    | `cvx add` - job extraction prompt    |
| `advise.md` | `cvx advise` - match analysis prompt |
| `build.md`  | `cvx build` - CV tailoring prompt    |

Template variables available: `{{.CVPath}}`, `{{.ReferencePath}}`

Reset to defaults with `cvx init -r`.

## Shell Completion

Generate shell completion scripts for your shell:

```bash
# Bash
cvx completion bash > /etc/bash_completion.d/cvx

# Zsh
cvx completion zsh > "${fpath[1]}/_cvx"

# Fish
cvx completion fish > ~/.config/fish/completions/cvx.fish
```

## Global Flags

| Flag            | Description                   |
| --------------- | ----------------------------- |
| `-q, --quiet`   | Suppress non-essential output |
| `-v, --verbose` | Enable debug logging          |
