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

| Flag        | Short | Description                                      |
| ----------- | ----- | ------------------------------------------------ |
| `--agent`   | `-a`  | CLI agent: claude, gemini                        |
| `--model`   | `-m`  | API model: claude-sonnet-4, gemini-2.5-flash etc |
| `--repo`    | `-r`  | GitHub repo (overrides config)                   |
| `--schema`  | `-s`  | Schema file path                                 |
| `--body`    | `-b`  | Read job posting from file                       |
| `--dry-run` |       | Extract only, don't create issue                 |

`--agent` and `--model` are mutually exclusive.

**Examples:**

```bash
cvx add https://company.com/job
cvx add https://company.com/job --dry-run
cvx add https://company.com/job -a gemini          # Gemini CLI
cvx add https://company.com/job -m claude-sonnet-4 # Claude API
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

| Flag            | Short | Description                                      |
| --------------- | ----- | ------------------------------------------------ |
| `--agent`       | `-a`  | CLI agent: claude, gemini                        |
| `--model`       | `-m`  | API model: claude-sonnet-4, gemini-2.5-flash etc |
| `--context`     | `-c`  | Additional context                               |
| `--interactive` | `-i`  | Interactive session                              |
| `--push`        | `-p`  | Post analysis to issue                           |

`--agent` and `--model` are mutually exclusive.

**Examples:**

```bash
cvx advise 42                         # Analyze issue #42
cvx advise 42 --push                  # Post as comment
cvx advise 42 -a gemini               # Gemini CLI
cvx advise 42 -m gemini-2.5-flash     # Gemini API
cvx advise 42 -c "Focus on backend"
cvx advise 42 -i                      # Interactive mode
```

---

## cvx build

Build tailored CV and cover letter for a job posting.

```bash
cvx build [issue-number] [flags]
```

| Flag            | Short | Description                                      |
| --------------- | ----- | ------------------------------------------------ |
| `--agent`       | `-a`  | CLI agent: claude, gemini                        |
| `--model`       | `-m`  | API model: claude-sonnet-4, gemini-2.5-flash etc |
| `--context`     | `-c`  | Feedback or additional context                   |
| `--interactive` | `-i`  | Interactive session (requires CLI agent)         |
| `--open`        | `-o`  | Open combined.pdf in VSCode after build          |
| `--commit`      |       | Commit changes on the issue branch               |
| `--push`        |       | Push commits to remote (requires --commit)       |

If issue-number is omitted, it's inferred from the current branch name.

**Examples:**

```bash
cvx build                            # Infer issue from branch
cvx build 42                         # Build for issue #42
cvx build -o                         # Build and open PDF
cvx build -c "emphasize Python"      # Continue with feedback
cvx build -i                         # Interactive session
cvx build --commit --push            # Build, commit, and push
```

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
