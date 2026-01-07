Keep only the following:

- `cvx new` should create a new project with a default config and files, similar to `zensical new`.
- `cvx add -m` should use model with api not cli, needs `job-ad.yml` issue template file.
- `cvx build` only produces ai-modified toml files and not pdfs (current). needs `schema.json`, `.env` file, -f `cvx.toml` file.
- `cvx version`

README.md instructions with `uv` dependency:
- `cvx new`
- `cvx add -m`
- `cvx build`

The rest goes into `justfile`:
no branching by `cvx`, just use `git` or `just init` with worktree.

- `just render`
- `just doctor`
- drop `src/*.tex` completely.
