# Configuration

## Config File

Located at `.cvx-config.yaml` in your repo root:

```yaml
repo: owner/repo
agent: claude
cv_path: src/cv.tex
reference_path: reference/
project: owner/1
```

## Settings

| Key | Description | Default |
|-----|-------------|---------|
| `repo` | GitHub repository (owner/repo) | Auto-detected |
| `agent` | AI agent to use | `claude` |
| `schema` | Job schema file path | Built-in |
| `cv_path` | CV file for advise/tailor | `src/cv.tex` |
| `reference_path` | Reference materials directory | `reference/` |
| `project` | GitHub Project (owner/number) | - |

## AI Agents

Priority order (first available is default):

| Agent | Notes |
|-------|-------|
| `claude` | Uses Claude Code CLI |
| `gemini` | Uses Gemini CLI |
| `claude-sonnet-4` | Requires `ANTHROPIC_API_KEY` |
| `claude-sonnet-4-5` | Requires `ANTHROPIC_API_KEY` |
| `claude-opus-4` | Requires `ANTHROPIC_API_KEY` |
| `claude-opus-4-5` | Requires `ANTHROPIC_API_KEY` |
| `gemini-2.5-flash` | Requires `GEMINI_API_KEY` |
| `gemini-2.5-pro` | Requires `GEMINI_API_KEY` |

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
  workflows/          # Workflow definitions (customizable)
  sessions/           # Agent session files
  matches/            # Match analysis outputs
```

## Customizing Workflows

AI prompts in `.cvx/workflows/` can be edited:

| File | Command | Purpose |
|------|---------|---------|
| `add.md` | `cvx add` | Job extraction prompt |
| `advise.md` | `cvx advise` | Match analysis prompt |
| `tailor.md` | `cvx tailor` | CV tailoring prompt |

**Template variables:**

- `{{.CVPath}}` - Path to CV file
- `{{.ReferencePath}}` - Path to reference directory

Reset to defaults: `cvx init -r`
