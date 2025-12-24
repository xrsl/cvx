# cvx

**Stop copying job descriptions into spreadsheets.** cvx uses AI to extract job details from any URL, tracks applications in GitHub Issues + Projects, and helps you tailor your CV and cover letter using LaTeX—all from the terminal.

**Why cvx?** Job hunting is tedious. You find a posting, copy details, track it somewhere, customize your resume, write a cover letter... cvx automates the boring parts so you can focus on what matters: landing the interview.

## Install

```bash
go install github.com/xrsl/cvx@latest
```

**Requirements:**

- [GitHub CLI](https://cli.github.com/) (`gh`) - installed and authenticated
- One of: [Claude CLI](https://github.com/anthropics/claude-code), [Gemini CLI](https://github.com/google-gemini/gemini-cli), or API key
- LaTeX: [BasicTeX](https://tug.org/mactex/morepackages.html) (light, recommended for Mac), [MacTeX](https://tug.org/mactex/), or [TeX Live](https://tug.org/texlive/) - for building PDFs

## Quick Start

```bash
cvx init                              # Setup wizard
cvx add https://company.com/job       # Add job posting
cvx advise 42                         # Analyze job-CV match
cvx tailor 42                         # Tailor CV/cover letter
cvx view 42                           # View submitted documents
```

## Commands

### `cvx add <url>`

Fetches job posting, extracts details with AI, creates GitHub issue.

```bash
cvx add https://company.com/job
cvx add https://company.com/job --dry-run     # extract only
cvx add https://company.com/job -a gemini
cvx add https://company.com/job --body        # use .cvx/body.md
cvx add https://company.com/job -b job.md     # use custom file
```

### `cvx list`

Lists all job applications with status, company, and deadline.

```bash
cvx list
cvx list --state closed   # show closed issues
cvx list --company google # filter by company
cvx list -r owner/repo    # specific repo
```

### `cvx advise <issue>`

Get career advice on job match quality.

```bash
cvx advise 42                    # Analyze issue #42
cvx advise 42 --push             # Post analysis as comment
cvx advise 42 -a gemini          # Use Gemini CLI
cvx advise 42 -i                 # Interactive session
cvx advise https://example.com/job
```

### `cvx tailor <issue>`

Tailor CV and cover letter interactively. The AI edits your LaTeX source files directly, then you can build PDFs with your TeX distribution.

```bash
cvx tailor 42                    # Start/resume tailoring session
cvx tailor 42 -a gemini          # Use Gemini CLI
cvx tailor 42 -c "Emphasize Python"
```

Sessions are shared per issue - `advise` and `tailor` continue the same conversation.

### `cvx view <issue>`

View submitted application documents.

```bash
cvx view 42                      # Open combined or CV PDF
cvx view 42 -l                   # Open cover letter
cvx view 42 -c                   # Open CV only
```

Opens the PDF from the git tag created when the application was submitted.

### `cvx rm <issue>`

Deletes an issue.

```bash
cvx rm 42
```

### `cvx init`

Interactive setup wizard.

```bash
cvx init                # interactive wizard
cvx init -q             # quiet mode with defaults
cvx init -r             # reset workflows to defaults
cvx init -c             # validate config resources
cvx init -d             # delete .cvx/ and config
```

## AI Agents

Priority order (first available is default):

| Agent               | Notes                        |
| ------------------- | ---------------------------- |
| `claude`            | Uses Claude Code CLI         |
| `gemini`            | Uses Gemini CLI              |
| `claude-sonnet-4`   | Requires `ANTHROPIC_API_KEY` |
| `claude-sonnet-4-5` | Requires `ANTHROPIC_API_KEY` |
| `claude-opus-4`     | Requires `ANTHROPIC_API_KEY` |
| `claude-opus-4-5`   | Requires `ANTHROPIC_API_KEY` |
| `gemini-2.5-flash`  | Requires `GEMINI_API_KEY`    |
| `gemini-2.5-pro`    | Requires `GEMINI_API_KEY`    |

## GitHub Project

cvx automatically creates a GitHub Project with:

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

Issues are automatically added to the project when created with `cvx add`.

## Config File

Located at `.cvx-config.yaml` in your repo root:

```yaml
repo: owner/repo
agent: claude
cv_path: src/cv.tex
reference_path: reference/
project: owner/1
```

The `reference_path` directory should contain your experience documentation, guidelines, and other reference materials used by `advise` and `tailor` commands.

Internal project IDs are cached in `.cvx/cache.yaml` (auto-managed).

## Customizing Workflows

AI prompts are stored in `.cvx/workflows/` and can be customized:

| File        | Used by                              |
| ----------- | ------------------------------------ |
| `add.md`    | `cvx add` - job extraction prompt    |
| `advise.md` | `cvx advise` - match analysis prompt |
| `tailor.md` | `cvx tailor` - CV tailoring prompt   |

Template variables available: `{{.CVPath}}`, `{{.ReferencePath}}`

Reset to defaults with `cvx init -r`.
