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
| `--text` | `-t` | Job text (skips URL fetch) |
| `--dry-run` | | Extract only, don't create issue |

**Examples:**

```bash
cvx add https://company.com/job
cvx add https://company.com/job --dry-run
cvx add https://company.com/job -a claude-cli:opus-4.5
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

## cvx status

Update application status.

```bash
cvx status <issue> <status>
cvx status --list
```

**Available statuses:**

- `to_be_applied`
- `applied`
- `interview`
- `offered`
- `accepted`
- `gone`
- `let_go`

**Examples:**

```bash
cvx status 42 applied
cvx status 42 interview
```

---

## cvx match

Analyze job-CV match quality using AI.

```bash
cvx match <issue-or-url> [flags]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--context` | `-c` | Additional context |
| `--interactive` | `-i` | Interactive session |
| `--push` | `-p` | Post analysis to issue |

**Examples:**

```bash
cvx match 42                    # Analyze issue #42
cvx match 42 --push             # Post as comment
cvx match 42 -c "Focus on backend"
cvx match 42 -i                 # Interactive mode
```

---

## cvx rm

Remove a job application.

```bash
cvx rm <issue>
```
