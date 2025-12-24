# Getting Started

## Installation

```bash
go install github.com/xrsl/cvx@latest
```

## Setup

Run the initialization wizard in your repository:

```bash
cvx init
```

The wizard will:

1. **Link your GitHub repo** - auto-detected from git remote
2. **Select AI agent** - Claude CLI, Gemini CLI, or API
3. **Set CV path** - for match analysis
4. **Set experience file** - reference for tailoring
5. **Create/link GitHub Project** - with job-tracking statuses

## First Job Application

Add a job posting:

```bash
cvx add https://company.com/careers/role
```

This will:

1. Fetch the job posting
2. Extract details using AI
3. Create a GitHub issue
4. Add to your project board

## View Applications

```bash
cvx list
```

## Workflow

```
cvx add <url>           # Add job posting
cvx list                # View all applications
cvx advise <issue>      # Analyze job-CV match
cvx tailor <issue>      # Tailor CV/cover letter
cvx rm <issue>          # Remove application
```
