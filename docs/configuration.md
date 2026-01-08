# Configuration

## Config File

The configuration file is `cvx.toml` in your repo root (created by `cvx init`).

```toml
[github]
repo = "owner/repo"
project = "owner/number"

[agent]
default = "claude"          # Default CLI agent for interactive mode

[cv]
source = "src/cv.toml"      # CV data source (toml)
output = "out/cv.pdf"       # Generated CV output
schema = "schema/schema.json"

[letter]
source = "src/letter.toml"  # Letter data source (toml)
output = "out/letter.pdf"   # Generated letter output
schema = "schema/schema.json"

[paths]
reference = "reference/"    # Reference materials for AI

[schema]
job_ad = ".github/ISSUE_TEMPLATE/job-ad-schema.yaml"
```

## Settings Reference

### GitHub Section

| Key       | Description                    | Default       |
| --------- | ------------------------------ | ------------- |
| `repo`    | GitHub repository (owner/repo) | Auto-detected |
| `project` | GitHub Project (owner/number)  | -             |

### Agent Section

| Key       | Description       | Default  |
| --------- | ----------------- | -------- |
| `default` | Default CLI agent | `claude` |

### CV & Letter Sections

| Key      | Description                | Default              |
| -------- | -------------------------- | -------------------- |
| `source` | Data file path (toml)      | `src/cv.toml`        |
| `output` | Generated PDF path         | `out/cv.pdf`         |
| `schema` | JSON schema for validation | `schema/schema.json` |

### Paths Section

| Key         | Description                | Default      |
| ----------- | -------------------------- | ------------ |
| `reference` | Reference materials folder | `reference/` |

### Schema Section

| Key      | Description              | Default                                     |
| -------- | ------------------------ | ------------------------------------------- |
| `job_ad` | Job ad extraction schema | `.github/ISSUE_TEMPLATE/job-ad-schema.yaml` |

## CLI Agents

Available agents for interactive mode:

| Agent    | Notes                                                     |
| -------- | --------------------------------------------------------- |
| `claude` | [Claude CLI](https://github.com/anthropics/claude-code)   |
| `gemini` | [Gemini CLI](https://github.com/google-gemini/gemini-cli) |

## API Models

Available models for agent mode (`cvx build -m`):

| Short Name     | Provider API Name      | Required Key        |
| -------------- | ---------------------- | ------------------- |
| `sonnet-4`     | claude-sonnet-4        | `ANTHROPIC_API_KEY` |
| `sonnet-4-5`   | claude-sonnet-4-5      | `ANTHROPIC_API_KEY` |
| `opus-4`       | claude-opus-4          | `ANTHROPIC_API_KEY` |
| `opus-4-5`     | claude-opus-4-5        | `ANTHROPIC_API_KEY` |
| `flash-2-5`    | gemini-2.5-flash       | `GEMINI_API_KEY`    |
| `pro-2-5`      | gemini-2.5-pro         | `GEMINI_API_KEY`    |
| `flash-3`      | gemini-3-flash-preview | `GEMINI_API_KEY`    |
| `pro-3`        | gemini-3-pro-preview   | `GEMINI_API_KEY`    |
| `gpt-oss-120b` | openai/gpt-oss-120b    | `GROQ_API_KEY`      |
| `qwen3-32b`    | qwen/qwen3-32b         | `GROQ_API_KEY`      |

## Environment Variables

### API Keys

| Variable            | Description            |
| ------------------- | ---------------------- |
| `ANTHROPIC_API_KEY` | For Claude API models  |
| `GEMINI_API_KEY`    | For Gemini API models  |
| `GROQ_API_KEY`      | For Groq-hosted models |
| `OPENAI_API_KEY`    | For OpenAI models      |

### Agent

| Variable            | Description                                |
| ------------------- | ------------------------------------------ |
| `AI_MODEL`          | Primary model (set by cvx automatically)   |
| `AI_FALLBACK_MODEL` | Fallback model (default: gemini-2.5-flash) |

### Environment File Loading

cvx loads `.env` files with the following priority:

1. `--env-file` flag (highest)
2. Current directory `.env`
3. Git worktree main repo `.env`
4. Parent directories `.env`
5. `~/.config/cvx/env` (lowest)

This enables API key management across git worktrees.

## GitHub Project

cvx creates a project with these fields:

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
cvx.toml              # User config (editable)
.cvx/
  cache.yaml          # Internal IDs (auto-managed)
  workflows/          # Workflow prompts (customizable)
  sessions/           # CLI session files
  matches/            # Match analysis outputs
```

## Customizing Workflows

AI prompts in `.cvx/workflows/` can be edited:

| File        | Command      | Purpose               |
| ----------- | ------------ | --------------------- |
| `add.md`    | `cvx add`    | Job extraction prompt |
| `advise.md` | `cvx advise` | Match analysis prompt |
| `build.md`  | `cvx build`  | CV tailoring prompt   |

**Template variables:**

- `{{.CVYAMLPath}}` - Path to CV data file (legacy name, works with TOML)
- `{{.ReferencePath}}` - Path to reference directory

Reset to defaults: delete `cvx.toml` and run `cvx init`
