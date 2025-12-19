# cvx

**cvx** is a CLI tool that automates job application tracking using AI agents. It fetches job postings, extracts key details (like company, role, location) using models like Gemini or Claude, and automatically creates structured GitHub issues to track your applications.

## Features

- **AI Extraction**: Uses Gemini or Claude to parse job descriptions into structured data.
- **GitHub Integration**: Automatically creates issues in your tracking repository using the `gh` CLI.
- **Custom Schemas**: specific parsing rules via GitHub issue template schemas.

## Installation

```bash
# Build from source
go install github.com/xrsl/cvx@latest
```

*Note: Requires [GitHub CLI](https://cli.github.com/) (`gh`) to be installed and authenticated.*

## Configuration

Set up your environment variables (e.g., `.env`):

```bash
GEMINI_API_KEY=your_key_here
# or
ANTHROPIC_API_KEY=your_key_here
```

Configure defaults:

```bash
cvx config set repo owner/repo_name
cvx config set schema /path/to/.github/ISSUE_TEMPLATE/job-ad.yml
```

## Usage

**Add a job application:**

```bash
cvx add https://jobs.example.com/software-engineer
```

**Dry run (extract only):**

```bash
cvx add https://jobs.example.com/software-engineer --dry-run
```

**Specify model:**

```bash
cvx add <url> -m gemini-pro
cvx add <url> -m claude-3-sonnet
```
