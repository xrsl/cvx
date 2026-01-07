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

**AI-powered CLI for CV tailoring and job application tracking.**

`cvx` uses AI to extract job details from any job posting URL, tracks your applications in GitHub Issues + Projects, and tailors your CV and cover letter — all from your terminal.

## Key Features

- **Multi-Provider AI**: Claude, Gemini, OpenAI, and Groq via [pydantic-ai](https://ai.pydantic.dev/)
- **Structured Output**: Schema-validated TOML with automatic retry and fallback
- **Interactive Mode**: Real-time editing with Claude CLI or Gemini CLI
- **GitHub Integration**: Issues, Projects, GraphQL API for full workflow automation
- **Polyglot Architecture**: Go CLI + embedded Python agent for AI operations

## Installation

```bash
go install github.com/xrsl/cvx@latest
```

## Requirements

- `git` + [GitHub CLI](https://cli.github.com/) (`gh`) - installed and authenticated
- [Claude CLI](https://github.com/anthropics/claude-code) or [Gemini CLI](https://github.com/google-gemini/gemini-cli)
- [Typst](https://typst.app/) - for PDF rendering
- [uv](https://docs.astral.sh/uv/) - for Python agent mode

## Quickstart

```bash
cvx init                              # Setup wizard
cvx add https://company.com/job       # Add job posting
cvx advise 42                         # Analyze job-CV match
cvx build 42                          # Build tailored CV/letter (interactive)
cvx build -m sonnet-4                 # Build with Python agent (API)
cvx approve 42                        # Commit, tag, push, update status
cvx view 42                           # View submitted documents
```

## Commands

| Command               | Description                                      |
| --------------------- | ------------------------------------------------ |
| `cvx add <url>`       | Extract job details with AI, create GitHub issue |
| `cvx list`            | List all job applications                        |
| `cvx advise <issue>`  | Analyze job-CV match quality                     |
| `cvx build [issue]`   | Build tailored CV and cover letter               |
| `cvx approve [issue]` | Commit, tag, push, update project status         |
| `cvx view <issue>`    | View submitted documents                         |
| `cvx rm <issue>`      | Remove job application                           |
| `cvx init`            | Initialize configuration                         |

## Build Modes

### Interactive CLI Mode (Default)

```bash
cvx build 42                # Start/resume session
cvx build -c "focus on ML"  # Add context
```

- Direct file editing via AI tool use
- Session persistence per issue
- Auto-detects `claude` or `gemini` CLI
- Uses the model configured within the CLI agent

### Python Agent Mode (API)

```bash
cvx build -m sonnet-4       # Claude API
cvx build -m flash-2-5      # Gemini API
cvx build -m qwen3-32b      # Groq API
```

- Calls AI provider APIs directly
- Structured TOML output validated against JSON Schema
- Multi-provider support with automatic fallback
- Uses [pydantic-ai](https://ai.pydantic.dev/) for structured generation

### Supported Models

| Short Name     | Provider  | API Name               |
| -------------- | --------- | ---------------------- |
| `sonnet-4`     | Anthropic | claude-sonnet-4        |
| `sonnet-4-5`   | Anthropic | claude-sonnet-4-5      |
| `opus-4`       | Anthropic | claude-opus-4          |
| `flash-2-5`    | Google    | gemini-2.5-flash       |
| `pro-2-5`      | Google    | gemini-2.5-pro         |
| `flash-3`      | Google    | gemini-3-flash-preview |
| `gpt-oss-120b` | Groq      | openai/gpt-oss-120b    |
| `qwen3-32b`    | Groq      | qwen/qwen3-32b         |

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                 Go CLI (cvx)                        │
│  Orchestration • GitHub API • Subprocess mgmt      │
└─────────────────────────────────────────────────────┘
                        │
         ┌──────────────┴──────────────┐
         ▼                             ▼
┌─────────────────────┐    ┌─────────────────────────┐
│  Interactive Mode   │    │  Python Agent Mode      │
│  claude/gemini CLI  │    │  pydantic-ai            │
│  Direct editing     │    │  Structured output      │
│  Session persist    │    │  Multi-provider         │
└─────────────────────┘    └─────────────────────────┘
                                       │
                        ┌──────────────┼──────────────┐
                        ▼              ▼              ▼
                   Claude API    Gemini API    Groq API
```

**Key Technical Decisions:**

1. **Go + Python Polyglot**: Go for CLI performance, Python for AI ecosystem (pydantic-ai)
2. **Subprocess over FFI**: Clean JSON stdin/stdout protocol for process isolation
3. **Embedded Agent**: Python agent embedded in Go binary, extracted at runtime via `uvx`
4. **Schema-Driven**: Single JSON Schema drives Pydantic models, TOML output, and IDE completion

## Configuration

Config file: `cvx.toml`

```toml
[github]
repo = "owner/repo"

[agent]
default = "claude"

[cv]
source = "src/cv.toml"
schema = "schema/schema.json"

[letter]
source = "src/letter.toml"

[paths]
reference = "reference/"
```

## Environment Variables

| Variable            | Description            |
| ------------------- | ---------------------- |
| `ANTHROPIC_API_KEY` | For Claude API models  |
| `GEMINI_API_KEY`    | For Gemini API models  |
| `GROQ_API_KEY`      | For Groq-hosted models |

Environment files are loaded with priority:

1. `--env-file` flag
2. Current directory `.env`
3. Git worktree main repo `.env`
4. Parent directories `.env`
5. `~/.config/cvx/env`

## Git Workflow

cvx uses branches for development and tags for archiving:

```bash
# Build creates/switches to issue branch
cvx build 42  # → branch: 42-company-role

# Approve commits, tags, and pushes
cvx approve 42  # → tag: 42-company-role-2025-01-07

# View retrieves from tag
cvx view 42  # Opens PDF from git tag
```

## Shell Completion

```bash
cvx completion bash > /etc/bash_completion.d/cvx
cvx completion zsh > "${fpath[1]}/_cvx"
cvx completion fish > ~/.config/fish/completions/cvx.fish
```

## Documentation

- [Getting Started](docs/getting-started.md)
- [Commands](docs/commands.md)
- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture.md)
- [Schema Reference](docs/schema.md)

## License

MIT
