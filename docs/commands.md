# Commands

## cvx init

Initialize cvx for the current repository.

```bash
cvx init
```

Creates `cvx.toml` config file and sets up the project structure.

**What the wizard configures:**

1. **GitHub repository** - auto-detected from git remote
2. **CLI agent** - Claude or Gemini CLI for interactive mode
3. **CV source path** - your CV data file (TOML)
4. **Letter source path** - your letter data file
5. **Reference directory** - additional context for AI
6. **Job ad schema** - template for extracted job data
7. **GitHub Project** - create new or link existing

---

## cvx add

Add a job application from a URL.

```bash
cvx add <url> [flags]
```

| Flag       | Short | Description                             |
| ---------- | ----- | --------------------------------------- |
| `--agent`  | `-a`  | CLI agent: `claude`, `gemini`           |
| `--model`  | `-m`  | Model for API mode (calls API directly) |
| `--repo`   | `-r`  | GitHub repo (overrides config)          |
| `--schema` | `-s`  | Schema file path                        |
| `--body`   | `-b`  | Read job posting from file              |

**Examples:**

```bash
cvx add https://company.com/job
cvx add https://company.com/job -a gemini          # Gemini CLI (uses agent's model)
cvx add https://company.com/job -m sonnet-4        # Claude API directly
cvx add https://company.com/job -m flash-2-5       # Gemini API directly
cvx add https://company.com/job --body             # Use .cvx/body.md
cvx add https://company.com/job -b job.md          # Use custom file
```

**Note:** Using `-a` (agent) uses the CLI tool with its configured model. Using `-m` (model) calls the API directly.

---

## cvx list

List all job applications.

```bash
cvx list [flags]
```

| Flag        | Short | Description                     |
| ----------- | ----- | ------------------------------- |
| `--state`   |       | Issue state (open\|closed\|all) |
| `--limit`   |       | Max issues to list (default 50) |
| `--company` |       | Filter by company name          |
| `--repo`    | `-r`  | GitHub repo (overrides config)  |

**Examples:**

```bash
cvx list
cvx list --state closed
cvx list --company google
```

---

## cvx advise

Get career advice on job match quality.

```bash
cvx advise <issue-or-url> [flags]
```

| Flag                | Short | Description                             |
| ------------------- | ----- | --------------------------------------- |
| `--agent`           | `-a`  | CLI agent: `claude`, `gemini`           |
| `--model`           | `-m`  | Model for API mode (calls API directly) |
| `--context`         | `-c`  | Additional context                      |
| `--post-as-comment` |       | Post analysis to issue                  |

**Examples:**

```bash
cvx advise 42                        # Analyze issue #42 (uses default CLI agent)
cvx advise 42 --post-as-comment      # Post as comment
cvx advise 42 -a gemini              # Use Gemini CLI (agent's model)
cvx advise 42 -m sonnet-4            # Use Claude API directly
cvx advise 42 -c "Focus on backend"  # Add context
cvx advise https://company.com/job   # Analyze URL directly
```

**Note:** Using `-a` (agent) uses the CLI tool with its configured model. Using `-m` (model) calls the API directly.

---

## cvx build

Build tailored CV and cover letter for a job posting.

```bash
cvx build [issue-number] [flags]
```

### Build Modes

`cvx build` supports two modes:

#### 1. Interactive CLI Mode (Default)

Auto-detects `claude` or `gemini` CLI:

- Direct file editing with AI tool use
- Session persistence per issue
- Resume previous sessions
- Uses the model configured within the CLI agent

```bash
cvx build                        # Interactive, infer issue from branch
cvx build 42                     # Interactive for issue #42
cvx build -c "focus on ML"       # Interactive with context
```

#### 2. Python Agent Mode (API)

Uses `-m` flag to call AI provider APIs directly:

- Structured TOML output
- Validates against JSON Schema using Pydantic
- Multi-provider support (Claude, Gemini, OpenAI, Groq)
- Automatic fallback on failure

```bash
cvx build -m sonnet-4            # Claude API
cvx build -m flash-2-5           # Gemini API
cvx build -m qwen3-32b           # Groq API
```

### Flags

| Flag        | Short | Description                                    |
| ----------- | ----- | ---------------------------------------------- |
| `--model`   | `-m`  | Use Python agent (calls API directly)          |
| `--context` | `-c`  | Feedback or additional context                 |
| `--schema`  | `-s`  | Schema path (overrides config)                 |
| `--branch`  | `-b`  | Switch to issue branch (creates if not exists) |

If issue-number is omitted, it's inferred from the current branch name.

### Supported Models

| Short Name     | Full API Name          |
| -------------- | ---------------------- |
| `sonnet-4`     | claude-sonnet-4        |
| `sonnet-4-5`   | claude-sonnet-4-5      |
| `opus-4`       | claude-opus-4          |
| `opus-4-5`     | claude-opus-4-5        |
| `flash-2-5`    | gemini-2.5-flash       |
| `pro-2-5`      | gemini-2.5-pro         |
| `flash-3`      | gemini-3-flash-preview |
| `pro-3`        | gemini-3-pro-preview   |
| `gpt-oss-120b` | openai/gpt-oss-120b    |
| `qwen3-32b`    | qwen/qwen3-32b         |

---

## cvx approve

Approve and finalize the tailored application.

```bash
cvx approve [issue-number]
```

This command:

1. Commits changes with message "Tailored application for [Company] [Role]"
2. Creates a git tag: `<issue>-<company>-<role>-YYYY-MM-DD`
3. Pushes the tag to origin
4. Updates GitHub project status to "Applied"
5. Sets AppliedDate field

If issue-number is omitted, it's inferred from the current branch name.

**Examples:**

```bash
cvx approve                          # Infer issue from branch
cvx approve 42                       # Approve issue #42
```

---

## cvx view

View submitted application documents.

```bash
cvx view <issue> [flags]
```

| Flag       | Short | Description       |
| ---------- | ----- | ----------------- |
| `--letter` | `-l`  | Open cover letter |
| `--cv`     | `-c`  | Open CV only      |

Opens the PDF from the git tag created when the application was submitted.

**Examples:**

```bash
cvx view 42                      # Open combined or CV
cvx view 42 -l                   # Open cover letter
cvx view 42 -c                   # Open CV only
```

---

## cvx rm

Remove a job application.

```bash
cvx rm <issue> [flags]
```

| Flag     | Short | Description                    |
| -------- | ----- | ------------------------------ |
| `--repo` | `-r`  | GitHub repo (overrides config) |

**Examples:**

```bash
cvx rm 123
cvx rm 123 -r owner/repo
```

---

## cvx completion

Generate shell completion scripts.

```bash
cvx completion [bash|zsh|fish|powershell]
```

**Examples:**

```bash
# Bash
cvx completion bash > /etc/bash_completion.d/cvx

# Zsh
cvx completion zsh > "${fpath[1]}/_cvx"

# Fish
cvx completion fish > ~/.config/fish/completions/cvx.fish
```

---

## Global Flags

| Flag         | Short | Description                                     |
| ------------ | ----- | ----------------------------------------------- |
| `--quiet`    | `-q`  | Suppress non-essential output                   |
| `--verbose`  | `-v`  | Enable debug logging                            |
| `--env-file` | `-e`  | Path to .env file (overrides default locations) |
