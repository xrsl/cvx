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

### `cvx status <issue> <status>`

Updates application status.

```bash
cvx status 42 applied
cvx status 42 interview
cvx status --list         # show available statuses
```

Available statuses: `to_be_applied`, `applied`, `interview`, `offered`, `accepted`, `gone`, `let_go`

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
| `claude-cli` | Uses Claude CLI (free with Claude subscription) |
| `claude-cli:opus-4.5` | Specific Claude agent via CLI |
| `claude-cli:sonnet-4` | Specific Claude agent via CLI |
| `gemini-cli` | Uses Gemini CLI |
| `gemini-2.5-flash` | Requires `GEMINI_API_KEY` |
| `claude-sonnet-4` | Requires `ANTHROPIC_API_KEY` |

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
schema: ""  # uses bundled default
project:
  number: 1
  owner: owner  # optional, inferred from repo
```

Internal project IDs are cached in `.cvx/cache.yaml` (auto-managed).
