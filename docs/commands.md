# Commands

## cvx init

Initialize cvx for the current repository.

```bash
cvx init [flags]
```

| Flag                | Short | Description                     |
| ------------------- | ----- | ------------------------------- |
| `--quiet`           | `-q`  | Non-interactive with defaults   |
| `--reset-workflows` | `-r`  | Reset workflows to defaults     |
| `--check`           | `-c`  | Validate config resources exist |
| `--delete`          | `-d`  | Remove .cvx/ and config file    |

Creates `.cvx-config.yaml` and `.cvx/` directory structure.

---

## cvx add

Add a job application from a URL.

```bash
cvx add <url> [flags]
```

| Flag        | Short | Description                                                               |
| ----------- | ----- | ------------------------------------------------------------------------- |
| `--agent`   | `-a`  | AI agent: claude-code, gemini-cli, api                                    |
| `--model`   | `-m`  | Model: sonnet-4, sonnet-4-5, opus-4, opus-4-5, flash, pro, flash-3, pro-3 |
| `--repo`    | `-r`  | GitHub repo (overrides config)                                            |
| `--schema`  | `-s`  | Schema file path                                                          |
| `--body`    | `-b`  | Read job posting from file                                                |
| `--dry-run` |       | Extract only, don't create issue                                          |

**Examples:**

```bash
cvx add https://company.com/job
cvx add https://company.com/job --dry-run
cvx add https://company.com/job -a gemini-cli       # Gemini AI agent
cvx add https://company.com/job -m sonnet-4         # Claude AI agent with sonnet-4 model
cvx add https://company.com/job -a api -m flash     # Gemini API directly with flash model
```

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

| Flag            | Short | Description                                                               |
| --------------- | ----- | ------------------------------------------------------------------------- |
| `--agent`       | `-a`  | AI agent: claude-code, gemini-cli, api                                    |
| `--model`       | `-m`  | Model: sonnet-4, sonnet-4-5, opus-4, opus-4-5, flash, pro, flash-3, pro-3 |
| `--context`     | `-c`  | Additional context                                                        |
| `--interactive` | `-i`  | Interactive session                                                       |
| `--push`        | `-p`  | Post analysis to issue                                                    |

**Examples:**

```bash
cvx advise 42                        # Analyze issue #42
cvx advise 42 --push                 # Post as comment
cvx advise 42 -a gemini-cli          # Gemini AI agent
cvx advise 42 -m sonnet-4            # Claude AI agent with sonnet-4 model
cvx advise 42 -a api -m flash        # Gemini API directly with flash model
cvx advise 42 -c "Focus on backend"
cvx advise 42 -i                     # Interactive mode
```

---

## cvx build

Build tailored CV and cover letter for a job posting.

```bash
cvx build [issue-number] [flags]
```

### Build Modes

`cvx build` supports three different modes:

#### 1. Python Agent Mode (Recommended)

Triggered when using `-m` flag **without** `--call-api-directly`. This mode:

- Uses structured YAML output (`src/cv.yaml`, `src/letter.yaml`)
- Validates output against `schema/schema.json`
- Provides automatic caching and multi-provider fallback
- Runs via `uv` (Astral's Python package manager)
- Requires: [uv](https://docs.astral.sh/uv/) installed

```bash
cvx build -m claude-sonnet-4         # Use Python agent with Claude
cvx build -m gemini-2.5-flash        # Use Python agent with Gemini
cvx build -m sonnet-4 --dry-run      # Preview without calling AI
cvx build -m sonnet-4 --no-cache     # Skip cache
```

**How it works:**

1. Fetches job posting from GitHub issue
2. Reads current CV and letter from YAML files
3. Computes cache key (job + CV + letter + schema + model)
4. Checks cache or calls Python agent via `uvx`
5. Python agent generates structured JSON using Pydantic AI
6. Validates output against schema
7. Writes updated YAML files

#### 2. CLI Agent Mode

Default mode when no `-m` flag is provided. Uses Claude Code or Gemini CLI:

- Edits LaTeX files directly (`src/cv.tex`, `src/letter.tex`)
- Supports interactive sessions with `-i`
- Session persistence per issue

```bash
cvx build                            # Use default CLI agent
cvx build -a claude                  # Use Claude CLI
cvx build -a gemini                  # Use Gemini CLI
cvx build -i                         # Interactive session
cvx build -c "emphasize Python"      # Continue with feedback
```

#### 3. Direct API Mode (Legacy)

Triggered with `--call-api-directly` flag. Directly calls AI APIs:

```bash
cvx build -m sonnet-4 --call-api-directly
cvx build -m flash --call-api-directly
```

### Flags

| Flag                  | Short | Description                                                               |
| --------------------- | ----- | ------------------------------------------------------------------------- |
| `--agent`             | `-a`  | CLI agent: claude-code, gemini-cli                                        |
| `--model`             | `-m`  | Model: sonnet-4, sonnet-4-5, opus-4, opus-4-5, flash, pro, flash-3, pro-3 |
| `--call-api-directly` |       | Use direct API mode (requires `--model`)                                  |
| `--context`           | `-c`  | Feedback or additional context                                            |
| `--interactive`       | `-i`  | Interactive session (CLI agent mode only)                                 |
| `--open`              | `-o`  | Open combined.pdf in VSCode after build                                   |
| `--commit`            |       | Commit changes on the issue branch                                        |
| `--push`              |       | Push commits to remote (requires `--commit`)                              |
| `--dry-run`           |       | Print plan without calling agent (Python agent mode only)                 |
| `--no-cache`          |       | Skip cache read/write (Python agent mode only)                            |

If issue-number is omitted, it's inferred from the current branch name.

### Examples

**Python Agent Mode:**

```bash
cvx build -m claude-sonnet-4         # Use Python agent with Claude
cvx build -m gemini-2.5-flash        # Use Python agent with Gemini
cvx build -m sonnet-4 --dry-run      # Preview cache key and plan
cvx build -m sonnet-4 --no-cache     # Force fresh AI generation
cvx build -m flash --commit --push   # Build, commit, and push
```

**CLI Agent Mode:**

```bash
cvx build                            # Infer issue from branch
cvx build 42                         # Build for issue #42
cvx build -i                         # Interactive session
cvx build -a claude                  # Use Claude CLI
cvx build -a gemini                  # Use Gemini CLI
cvx build -c "emphasize Python"      # Continue with feedback
cvx build --commit --push            # Build, commit, and push
```

**Direct API Mode:**

```bash
cvx build -m sonnet-4 --call-api-directly
cvx build -m flash --call-api-directly
```

### Python Agent Schema

When using Python agent mode, CV and letter data conform to the JSON schema in `schema/schema.json`:

**CV Structure:**

- `name`, `email`, `phone`, `location`
- `headline`, `expertise_tags`
- `social_networks` (LinkedIn, GitHub, etc.)
- `sections`:
  - `experience` - Work history with highlights
  - `education` - Academic background
  - `skills` - Technical and soft skills
  - `publications` - Research papers
  - `honors_and_awards` - Achievements
  - `summary` - Professional summary

**Letter Structure:**

- `sender` - Your contact information
- `recipient` - Company/hiring manager details
- `metadata` - Date, position applied
- `content`:
  - `salutation` - Opening greeting
  - `opening` - Introduction paragraph
  - `body` - Main paragraphs
  - `closing` - Conclusion and sign-off

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

Opens the PDF from the git tag created when the application was submitted. By default, opens the combined PDF (CV + letter) if available, otherwise falls back to CV.

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
