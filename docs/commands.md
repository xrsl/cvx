# Commands

## cvx init

Initialize cvx for the current repository.

```bash
cvx init
```

Creates `.cvx-config.yaml` and `.cvx/` directory structure.

---

## cvx add

Add a job application from a URL.

```bash
cvx add <url> [flags]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | `-a` | AI agent (overrides config) |
| `--repo` | `-r` | GitHub repo (overrides config) |
| `--schema` | `-s` | Schema file path |
| `--dry-run` | | Extract only, don't create issue |

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

| Flag | Short | Description |
|------|-------|-------------|
| `--repo` | `-r` | GitHub repo (overrides config) |

---

## cvx advise

Get career advice on job match quality.

```bash
cvx advise <issue-or-url> [flags]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--context` | `-c` | Additional context |
| `--interactive` | `-i` | Interactive session |
| `--push` | `-p` | Post analysis to issue |

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

| Flag | Short | Description |
|------|-------|-------------|
| `--context` | `-c` | Additional context |

Sessions are shared per issue - `advise` and `tailor` continue the same conversation.

**Examples:**

```bash
cvx tailor 42                    # Start/resume session
cvx tailor 42 -c "Emphasize Python"
```

---

## cvx rm

Remove a job application.

```bash
cvx rm <issue>
```
