# cvx

`cvx` uses AI to extract job details from any job posting URL, tracks your applications in GitHub Issues + Projects, and helps you tailor your CV and cover letter using LaTeX — all from your terminal.

## What it does

- **Extracts job details** from URLs using AI agents (Claude Code or Gemini CLI)
- **Creates GitHub Issues** with structured job information using a custimizable template (default `job-ad-schema.yaml`).
- **Tracks applications in a GitHub Project** with status, company, and deadlines
- **Analyzes job-CV match** quality with AI agents
- **Tailors CV and cover letter** with AI agents by editing your LaTeX source files

## Quick Example

```bash
# Initialize in your repo
cvx init

# Add a job posting
cvx add https://company.com/careers/software-engineer

# Analyze match quality
cvx advise 42 -a gemini

# Build tailored CV/cover letter
cvx build 42

# Approve and finalize
cvx approve 42

# View submitted documents
cvx view 42
```

## Branching and Tagging strategy

`cvx` uses git branches for building and git tags for archiving submitted applications:

```bash
# Build CV/letter (auto-creates branch 42-company-role)
cvx build 42

# Once satisfied, approve to commit, tag, and push
cvx approve 42

# This creates git tag in the following format:
# 42-company-role-2025-12-18

# Later, view what you submitted
cvx view 42
```

**Branch format**: `{issue}-{company}-{role}`

**Tag format**: `{issue}-{company}-{role}-{application-date}`

This keeps a permanent record of exactly what you sent to each company.

## Requirements

- `git` and [GitHub CLI](https://cli.github.com/) (`gh`) - installed and authenticated
- One of: [Claude Code CLI](https://github.com/anthropics/claude-code), [Gemini CLI](https://github.com/google-gemini/gemini-cli), or an API key (`ANTROPIC_API_KEY` or `GEMINI_API_KEY`).
- LaTeX: [BasicTeX](https://tug.org/mactex/morepackages.html) (light, recommended for Mac), [MacTeX](https://tug.org/mactex/), or [TeX Live](https://tug.org/texlive/) - for building PDFs
