# Getting Started

## Installation

```bash
go install github.com/xrsl/cvx@latest
```

## Setup

Run the initialization wizard in your repository:

```bash
cvx init
```

The wizard will:

1. **Link your GitHub repo** - auto-detected from git remote
2. **Select AI agent** - Claude CLI, Gemini CLI, or API
3. **Set CV path** - for match analysis
4. **Set experience file** - reference for tailoring
5. **Create/link GitHub Project** - with job-tracking statuses

## First Job Application

Add a job posting:

```bash
cvx add https://company.com/careers/role
```

This will:

1. Fetch the job posting
2. Extract details using AI
3. Create a GitHub issue
4. Add to your project board

## View Applications

```bash
cvx list
```

## Workflow

```
cvx init                # Initialize project
cvx add <url>           # Add job posting
cvx advise <issue>      # Analyze job-CV match
cvx build [issue]       # Build tailored CV/cover letter
cvx approve [issue]     # Commit, tag, push
cvx view <issue>        # View submitted documents
```

## Build Modes

`cvx build` supports multiple modes:

### Python Agent Mode (Recommended)

Use structured YAML output with schema validation:

```bash
cvx build -m claude-sonnet-4      # Python agent with Claude
cvx build -m gemini-2.5-flash     # Python agent with Gemini
```

**Requirements:**

- [uv](https://docs.astral.sh/uv/) installed
- YAML files: `src/cv.yaml`, `src/letter.yaml`
- Schema: `schema/schema.json`

**Benefits:**

- Structured, validated output
- Automatic caching
- Multi-provider fallback

### CLI Agent Mode

Use interactive or headless CLI agents:

```bash
cvx build                # Default CLI agent
cvx build -a claude      # Claude CLI
cvx build -i             # Interactive mode
```

**Benefits:**

- Direct LaTeX editing
- Interactive sessions
- Flexible document generation

## Other Commands

```
cvx list                # View all applications
cvx rm <issue>          # Remove application
```
