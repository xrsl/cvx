# cvx

CLI tool for tracking job applications. Uses AI to extract job details and GitHub Issues + Projects for tracking.

## Install

```bash
go install github.com/xrsl/cvx@latest
```

**Requirements:**
- [GitHub CLI](https://cli.github.com/) (`gh`) - installed and authenticated
- One of: [Claude CLI](https://github.com/anthropics/claude-code), [Gemini CLI](https://github.com/google-gemini/gemini-cli), or API key

## Quick Start

```bash
cvx config
```

This runs the setup wizard:
- Links your GitHub repo
- Selects AI agent
- Creates a GitHub Project with job-tracking statuses

Then add jobs:

```bash
cvx add https://company.com/careers/role
```

## Commands

### `cvx add <url>`

Fetches job posting, extracts details with AI, creates GitHub issue.

```bash
cvx add https://company.com/job
cvx add https://company.com/job --dry-run    # extract only
cvx add https://company.com/job -a claude-cli:opus-4.5
```

### `cvx list`

Lists all job applications with status, company, and deadline.

```bash
cvx list
cvx list -r owner/repo    # specific repo
```

### `cvx advise <issue>`

Get career advice on job match quality.

```bash
cvx advise 42                    # Analyze issue #42
cvx advise 42 --push             # Post analysis as comment
cvx advise 42 -i                 # Interactive session
cvx advise https://example.com/job
```

### `cvx tailor <issue>`

Tailor CV and cover letter interactively.

```bash
cvx tailor 42                    # Start/resume tailoring session
cvx tailor 42 -c "Emphasize Python"
```

Sessions are shared per issue - `advise` and `tailor` continue the same conversation.

### `cvx rm <issue>`

Deletes an issue.

```bash
cvx rm 42
```

### `cvx config`

Interactive setup wizard. Also supports direct access:

```bash
cvx config              # wizard
cvx config list         # show all settings
cvx config get agent    # get value
cvx config set agent claude-cli:opus-4.5
```

## AI Agents

Priority order (first available is default):

| Agent | Notes |
|-------|-------|
| `claude-cli` | Uses Claude Code CLI |
| `gemini-cli` | Uses Gemini CLI |
| `claude-sonnet-4` | Requires `ANTHROPIC_API_KEY` |
| `claude-sonnet-4-5` | Requires `ANTHROPIC_API_KEY` |
| `claude-opus-4` | Requires `ANTHROPIC_API_KEY` |
| `claude-opus-4-5` | Requires `ANTHROPIC_API_KEY` |
| `gemini-2.5-flash` | Requires `GEMINI_API_KEY` |
| `gemini-2.5-pro` | Requires `GEMINI_API_KEY` |

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
agent: claude-cli
cv_path: src/cv.tex
reference_path: reference/
project:
  number: 1
  owner: owner  # optional, inferred from repo
```

The `reference_path` directory should contain your experience documentation, guidelines, and other reference materials used by `advise` and `tailor` commands.

Internal project IDs are cached in `.cvx/cache.yaml` (auto-managed).
