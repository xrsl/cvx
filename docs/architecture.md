# Architecture

## Overview

`cvx` is a Go CLI that orchestrates AI-powered CV tailoring and job application workflows. It uses a **polyglot architecture**:

- **Go**: CLI, orchestration, caching, GitHub integration
- **Python**: AI agent with multi-provider support and schema validation
- **LaTeX/Typst**: Document rendering

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Interface                          │
│                    cvx CLI (Go + Cobra)                        │
└─────────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Interactive    │  │  Agent Mode     │  │   GitHub API    │
│  CLI Mode       │  │                 │  │   Integration   │
│                 │  │                 │  │                 │
│  claude/gemini  │  │  pydantic-ai    │  │  Issues, Projects│
│  Direct editing │  │  Structured out │  │  GraphQL/REST   │
└─────────────────┘  └─────────────────┘  └─────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                      AI Provider Layer                          │
│   Claude │ Gemini │ OpenAI │ Groq │ Claude CLI │ Gemini CLI    │
└─────────────────────────────────────────────────────────────────┘
```

## Build Modes

### Interactive CLI Mode (Default)

Invoked when running `cvx build` without `-m` flag.

```
┌────────────┐
│ User       │
│ cvx build  │
└─────┬──────┘
      │
      ▼
┌───────────────────────────┐
│ Go CLI (cvx)              │
│ - Check for existing      │
│   session (.cvx/sessions) │
│ - Fetch job from GH issue │
│ - Build prompt from       │
│   .cvx/workflows/build.md │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ claude / gemini CLI       │
│ - Tool use (Read/Edit)    │
│ - Session persistence     │
│ - Interactive mode        │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ LaTeX/Typst files         │
│ (modified directly)       │
└───────────────────────────┘
```

**Key Features:**

- **Session Persistence**: Resume sessions per issue
- **Direct Tool Use**: AI directly edits files
- **Interactive Mode**: Full CLI access
- **Auto-detection**: Detects claude or gemini CLI

### Agent Mode

Invoked when running `cvx build -m <model>`.

```
┌────────────┐
│ User       │
│ cvx build  │
│ -m model   │
└─────┬──────┘
      │
      ▼
┌────────────────────────────┐
│ Go CLI (cvx)               │
│ - Fetch job from GH issue  │
│ - Read cv.toml/letter.toml │
│ - Set AI_MODEL env var     │
└─────┬──────────────────────┘
      │
      ▼
┌────────────────────────────┐
│ uvx --from <agent> cvx-agent│
│ (isolated Python env)      │
│ - pydantic-ai              │
│ - Multi-provider support   │
│ - Structured output        │
└─────┬──────────────────────┘
      │ JSON stdin/stdout
      ▼
┌───────────────────────────┐
│ AI Provider               │
│ (Claude/Gemini/OpenAI/    │
│  Groq)                    │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ Pydantic Validation       │
│ - Model conformance       │
│ - Automatic retry         │
│ - Provider fallback       │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ Go CLI (cvx)              │
│ - Write cv.toml           │
│ - Write letter.toml       │
│ - Auto-format with tombi  │
└───────────────────────────┘
```

**Key Features:**

- **Structured Output**: YAML/TOML conforming to JSON Schema
- **Multi-Provider**: Claude, Gemini, OpenAI, Groq via pydantic-ai
- **Automatic Fallback**: Retry with fallback model on failure
- **Isolation**: Agent runs via `uvx` in isolated environment

## Agent Design

The agent is an **embedded subprocess** designed for:

### Multi-Provider AI Support

```python
def get_agent(model_name: str) -> Agent:
    if "gemini" in model_name or "flash" in model_name:
        model = GoogleModel(model_name)
    elif "claude" in model_name or "sonnet" in model_name:
        model = AnthropicModel(model_name)
    elif "openai/" in model_name or "qwen/" in model_name:
        model = GroqModel(model_name)
    else:
        model = OpenAIChatModel(model_name)

    return Agent(model=model, ...)
```

### Structured Output with Pydantic

```python
from pydantic_ai import Agent
from cvx_agent.models import Model

result = agent.run_sync(
    user_prompt=prompt,
    output_type=Model  # Pydantic model from schema.json
)
```

### Automatic Retry & Fallback

```python
PRIMARY_MODEL = os.getenv("AI_MODEL", "claude-haiku-4-5")
FALLBACK_MODEL = os.getenv("AI_FALLBACK_MODEL", "gemini-2.5-flash")

for model_name in [PRIMARY_MODEL, FALLBACK_MODEL]:
    for attempt in range(max_retries):
        try:
            result = agent.run_sync(...)
            return result.output.model_dump()
        except Exception:
            continue
```

## Subprocess Communication

The Go CLI and agent communicate via JSON over stdin/stdout:

```go
// Go: Send input
input := map[string]interface{}{
    "job_posting": issueBody,
    "cv":          cvData,
    "letter":      letterData,
}
json.NewEncoder(stdin).Encode(input)

// Go: Receive output
var output struct {
    CV     map[string]interface{} `json:"cv"`
    Letter map[string]interface{} `json:"letter"`
}
json.NewDecoder(stdout).Decode(&output)
```

## Environment Variable Loading

`cvx` implements priority-based environment loading:

```
Priority (highest to lowest):
1. --env-file flag (explicit)
2. Current directory .env
3. Git worktree main repo .env
4. Parent directories .env
5. ~/.config/cvx/env (user-level)
```

This enables secure API key management across worktrees and CI/CD.

## File Structure

```
.
├── cvx.toml                # User configuration (editable)
├── .cvx/
│   ├── cache.yaml          # GitHub project ID cache (auto-managed)
│   ├── workflows/          # AI prompts (customizable)
│   │   ├── add.md          # Job extraction prompt
│   │   ├── advise.md       # Match analysis prompt
│   │   └── build.md        # CV tailoring prompt
│   ├── sessions/           # CLI session files
│   └── matches/            # Match analysis outputs
├── schema/
│   └── schema.json         # CV/letter JSON schema
├── src/
│   ├── cv.toml             # Structured CV data
│   ├── letter.toml         # Structured letter data
│   ├── cv.tex              # LaTeX CV template
│   └── letter.tex          # LaTeX letter template
└── out/
    └── combined.pdf        # Generated PDF
```

## Agent Embedding

The agent is **embedded** in the Go binary:

```go
//go:embed agent/*
var agentFS embed.FS
```

At runtime, it's extracted to `~/.cache/cvx/agent/`:

```go
func extractAgentToCache() (string, error) {
    cacheDir := filepath.Join(os.UserCacheDir(), "cvx", "agent")
    // Extract if not present or version changed
    fs.WalkDir(agentFS, "agent", func(path string, d fs.DirEntry) {
        // Copy files to cache
    })
    return cacheDir, nil
}
```

## Model Generation

Pydantic models are auto-generated from `schema/schema.json`:

```bash
datamodel-codegen --input schema/schema.json --output agent/cvx_agent/models.py
```

The Go CLI regenerates models when schema changes:

```go
func regenerateModels(agentDir, schemaPath string) {
    schemaHash := sha256(schemaContent)
    if schemaHash != cachedHash {
        exec.Command("uv", "run", "datamodel-codegen", ...)
    }
}
```

## Key Technical Decisions

1. **Go + Python Polyglot**: Go for CLI performance and GitHub integration, Python for AI ecosystem compatibility (pydantic-ai, LangChain-compatible)

2. **Subprocess over FFI**: JSON stdin/stdout for clean process isolation and error handling

3. **uv/uvx for Python**: Fast, reliable Python environment management without system dependencies

4. **Embedded Agent**: Single binary distribution with embedded agent

5. **Schema-Driven**: Single JSON Schema drives Pydantic validation, YAML/TOML output, and IDE completion
