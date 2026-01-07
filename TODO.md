Keep only the following:

- `cvx new` should create a new project with a default config and files, similar to `zensical new`.
- `cvx add -m` should use model with api not cli.
- `cvx build` only produces modified toml files and not pdfs (current).
- `cvx doctor`
- `cvx version`

The rest goes into `justfile`:

- `just render`
- drop `src/*.tex` completely.
