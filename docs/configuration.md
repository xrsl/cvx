# Configuration

## Config File

Located at `.cvx-config.yaml` in your repo root:

```yaml
repo: owner/repo
agent: claude-cli
schema: ""
cv_path: src/cv.tex
experience_path: reference/EXPERIENCE.md
project:
  number: 1
  owner: owner
```

## Settings

| Key | Description | Default |
|-----|-------------|---------|
| `repo` | GitHub repository (owner/repo) | Auto-detected |
| `agent` | AI agent to use | `claude-cli` |
| `schema` | Job schema file path | Built-in |
| `cv_path` | CV file for match analysis | `src/cv.tex` |
| `experience_path` | Experience reference file | `reference/EXPERIENCE.md` |
| `project.number` | GitHub Project number | - |
| `project.owner` | Project owner | From repo |

## AI Agents

Priority order (first available is default):

| Agent | Notes |
|-------|-------|
| `claude-cli` | Uses Claude CLI |
| `claude-cli:opus-4.5` | Specific Claude agent |
| `claude-cli:sonnet-4` | Specific Claude agent |
| `gemini-cli` | Uses Gemini CLI |
| `gemini-2.5-flash` | Requires `GEMINI_API_KEY` |
| `claude-sonnet-4` | Requires `ANTHROPIC_API_KEY` |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | For Claude API agents |
| `GEMINI_API_KEY` | For Gemini API agents |
| `CVX_*` | Override any config (e.g., `CVX_AGENT`) |

## GitHub Project

cvx creates a project with:

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

## Directory Structure

```
.cvx-config.yaml      # User config
.cvx/
  cache.yaml          # Internal IDs (auto-managed)
  workflows/          # Workflow definitions
  sessions/           # Agent session files
  matches/            # Match analysis outputs
```
