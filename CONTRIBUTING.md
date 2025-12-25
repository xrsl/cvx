# Contributing

## Dev Environment

1. Run `make` to build and install the cli.
2. Install [`prek`](https://prek.j178.dev/) via `uv tool install prek`. Then, run `prek run --all-files` to ensure linting passes.

## Pull Requests

- Keep changes focused and atomic
- Ensure all tests pass
- Follow existing code style

## Understanding Claude Code Usage

Search and find the session that started with "fix one".

```
grep -l "fix one" ~/.claude/projects/**/*.jsonl
grep -l "fix one" ~/.claude/projects/**/*.jsonl | xargs -n 1 basename -s .jsonl
```

Then see session usage using [`ccusage`](https://ccusage.com/) with [`bunx`](https://bun.com/docs/installation#package-managers)

```
bunx ccusage session --id agent-a45e0e7
```

To list your most recent Claude Code session IDs and limit the output to the first N results (newest first), use this command:

```
ls -t ~/.claude/projects/**/*.jsonl | head -n 5 | xargs -n 1 basename -s .jsonl
```

# To test lint job in ci.yaml locally

```
golangci-lint run
```
