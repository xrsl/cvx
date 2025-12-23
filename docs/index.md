# cvx

CLI tool for tracking job applications using AI and GitHub.

## What it does

- **Extract job details** from URLs using AI (Claude or Gemini)
- **Create GitHub Issues** with structured job information
- **Track in GitHub Projects** with status, company, and deadlines
- **Analyze job-CV match** quality with AI agents

## Quick Example

```bash
# Initialize in your repo
cvx init

# Add a job posting
cvx add https://company.com/careers/software-engineer

# Analyze match quality
cvx match 42

# Update status
cvx status 42 applied
```

## Requirements

- [GitHub CLI](https://cli.github.com/) (`gh`) - installed and authenticated
- One of:
    - [Claude CLI](https://github.com/anthropics/claude-code)
    - [Gemini CLI](https://github.com/google-gemini/gemini-cli)
    - API key (`ANTHROPIC_API_KEY` or `GEMINI_API_KEY`)
