# Architecture

## Overview

`cvx` is a Go CLI tool that integrates with AI agents to automate job application workflows. It uses GitHub for tracking, LaTeX for document generation, and supports multiple AI providers.

## Build Modes

### Python Agent Mode (Recommended)

Used when running `cvx build -m <model>` (without `--call-api-directly`).

```
┌────────────┐
│ User       │
│ cvx build  │
│ -m model   │
└─────┬──────┘
      │
      ▼
┌───────────────────────────┐
│ Go CLI (cvx)              │
│ - Fetch job from GH issue │
│ - Read cv.yaml/letter.yaml│
│ - Compute cache hash      │
│ - Check cache (.cvx/)     │
└─────┬─────────────────────┘
      │ cache miss
      ▼
┌────────────────────────────┐
│ uvx --from <agent> cvx-agent│
│ (isolated Python env)      │
│ - pydantic-ai              │
│ - multi-provider support   │
│ - retry logic              │
└─────┬──────────────────────┘
      │ JSON stdin/stdout
      ▼
┌───────────────────────────┐
│ AI Provider               │
│ (Claude/Gemini/OpenAI)    │
│ Structured output         │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ Pydantic Validation       │
│ schema.json conformance   │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ Go CLI (cvx)              │
│ - Write cv.yaml           │
│ - Write letter.yaml       │
│ - Save to cache           │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ LaTeX Rendering           │
│ make combined             │
└───────────────────────────┘
```

**Key Features:**

- **Structured Output**: YAML files validated against JSON Schema
- **Caching**: SHA256 hash of (job + CV + letter + schema + model)
- **Reliability**: Multi-provider fallback, retry logic
- **Isolation**: Python agent runs in isolated environment via `uvx`

### CLI Agent Mode

Used when running `cvx build` or `cvx build -a <agent>`.

```
┌────────────┐
│ User       │
│ cvx build  │
│ -a claude  │
└─────┬──────┘
      │
      ▼
┌───────────────────────────┐
│ Go CLI (cvx)              │
│ - Fetch job from GH issue │
│ - Build prompt            │
│ - Check session cache     │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ claude / gemini CLI       │
│ - Direct file editing     │
│ - Tool use (Read/Edit)    │
│ - Interactive mode        │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ cv.tex / letter.tex       │
│ (modified directly)       │
└─────┬─────────────────────┘
      │
      ▼
┌───────────────────────────┐
│ LaTeX Rendering           │
│ make combined             │
└───────────────────────────┘
```

**Key Features:**

- **Interactive Sessions**: Full CLI access with `-i` flag
- **Session Persistence**: Resume sessions per issue
- **Direct Editing**: LaTeX files modified by AI agents
- **Flexibility**: Free-form document generation

## Data Flow

### Python Agent Mode

1. **Input Processing**:

   - Job posting fetched from GitHub issue
   - Current CV/letter read from `src/cv.yaml` and `src/letter.yaml`
   - Schema loaded from `schema/schema.json`

2. **Cache Check**:

   - Cache key: `SHA256(job + cv + letter + schema + model)`
   - Cache location: `.cvx/cache.yaml`
   - Skip with `--no-cache` flag

3. **AI Generation**:

   - Python agent called via `uvx --from <agent-dir> cvx-agent`
   - JSON input sent via stdin
   - Structured JSON output received via stdout
   - Pydantic validation against schema

4. **Output Writing**:
   - Validated data written to YAML files
   - Cache updated (unless `--no-cache`)
   - LaTeX rendering triggered

### Environment Variables

**Python Agent Mode:**

- `AI_MODEL` - Model to use (set automatically by cvx)
- `AI_FALLBACK_MODEL` - Fallback model (default: `gemini-2.5-flash`)
- `CVX_AGENT_CACHE` - Cache directory (default: `.cache`)
- `ANTHROPIC_API_KEY` - For Claude models
- `GEMINI_API_KEY` - For Gemini models
- `OPENAI_API_KEY` - For OpenAI models

## File Structure

```
.
├── .cvx/
│   ├── workflows/          # AI prompts (add.md, advise.md, build.md)
│   └── cache.yaml          # GitHub project ID cache
├── .cvx-config.yaml        # cvx configuration
├── schema/
│   └── schema.json         # CV/letter JSON schema
├── src/
│   ├── cv.yaml             # Structured CV data (Python agent)
│   ├── letter.yaml         # Structured letter data (Python agent)
│   ├── cv.tex              # LaTeX CV template
│   └── letter.tex          # LaTeX letter template
└── build/
    └── combined.pdf        # Generated PDF
```

## Python Agent Internals

Located in embedded `agent/` directory, extracted to `~/.cache/cvx/agent/`:

```
agent/
├── pyproject.toml          # uv project config
├── cvx_agent/
│   ├── __init__.py
│   ├── main.py            # Entry point (stdin/stdout)
│   ├── agents.py          # Core AI logic
│   └── models.py          # Pydantic models (from schema.json)
```

**Key Components:**

- **main.py**: Reads JSON from stdin, calls agent, writes to stdout
- **agents.py**:
  - Multi-provider support (Claude, Gemini, OpenAI)
  - Automatic fallback on failure
  - Retry logic (2 attempts per model)
  - Caching (SHA256 hash)
- **models.py**: Auto-generated from `schema/schema.json` using `datamodel-codegen`
