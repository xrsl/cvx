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

| Flag        | Short | Description                      |
| ----------- | ----- | -------------------------------- |
| `--agent`   | `-a`  | AI agent (overrides config)      |
| `--repo`    | `-r`  | GitHub repo (overrides config)   |
| `--schema`  | `-s`  | Schema file path                 |
| `--dry-run` |       | Extract only, don't create issue |

If `.cvx/body.md` exists with content, uses that instead of fetching URL.

**Examples:**

```bash
cvx add https://company.com/job
cvx add https://company.com/job --dry-run
cvx add https://company.com/job -a gemini
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

| Flag            | Short | Description            |
| --------------- | ----- | ---------------------- |
| `--context`     | `-c`  | Additional context     |
| `--interactive` | `-i`  | Interactive session    |
| `--push`        | `-p`  | Post analysis to issue |

**Examples:**

```bash
cvx advise 42                    # Analyze issue #42
cvx advise 42 --push             # Post as comment
cvx advise 42 -c "Focus on backend"
cvx advise 42 -i                 # Interactive mode
```

---

## cvx tailor

Tailor CV and cover letter interactively.

```bash
cvx tailor <issue> [flags]
```

| Flag        | Short | Description        |
| ----------- | ----- | ------------------ |
| `--context` | `-c`  | Additional context |

Sessions are shared per issue - `advise` and `tailor` continue the same conversation.

**Examples:**

```bash
cvx tailor 42                    # Start/resume session
cvx tailor 42 -c "Emphasize Python"
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
