# Contributing

Thanks for considering contributing to this project — we appreciate your time and effort.

## Table of contents
- How to contribute
- Reporting issues
- Branches & pull requests
- Running the project locally
- Tests & CI
- Code style & commit messages
- Code review
- Security

## How to contribute

1. Search existing issues before opening a new one.
2. Open an issue to discuss larger changes before implementing.
3. Fork the repo and create feature branches from `main`.

Keep changes small and focused; one feature or bugfix per branch/PR.

## Reporting issues

- Provide a clear title and description.
- Include steps to reproduce, expected vs actual behavior, and relevant logs or stack traces.
- If possible, include a small, self-contained reproduction.

## Branches & pull requests

- Branch naming: `feature/xyz`, `fix/bug-description`, or `chore/name`.
- Rebase or merge `main` into your branch to resolve conflicts before opening a PR.
- Open a PR against `main` and include a descriptive title and summary of changes.
- Link related issue numbers (e.g., `Fixes #123`).

PR checklist:
- Follow the coding style and include tests for new behavior.
- Keep the PR focused and include screenshots or logs if applicable.
- Add migration steps or configuration notes if the change requires them.
- Ensure automated CI checks pass before requesting a review.
- Require at least **2 approvals** from reviewers before merging.

## Running the project locally

This is a Go-based backend. Common commands:

```
# Run tests
go test ./...

# Start development stack using Docker Compose (dev config)
docker-compose -f docker-compose.dev.yml up --build

# Run the service directly
go run main.go
```

If you use Docker, ensure Docker Desktop is running and necessary environment files are present (for example, `app-sample.env`). Check the repository root for additional dev notes.

## Tests & CI

- Ensure `go test ./...` passes locally before opening a PR.
- Add unit tests for new features and bug fixes.
- If your change affects migrations, include tests that exercise those changes where feasible.

Continuous integration will run automated checks; address any failures reported by the CI in your branch.

## Code style & commit messages

- Follow Go idioms and formatting rules. Run `gofmt` on changed files.
- Commit messages should start with one of the conventional prefixes:

- `fix:` — bug fix
- `feat:` or `feature:` — new feature
- `docs:` — documentation only changes
- `chore:` — maintenance tasks
- `refactor:` — code change that neither fixes a bug nor adds a feature
- `test:` — adding or updating tests
- `perf:` — performance improvements
- `ci:` — continuous integration related
- `style:` — formatting, missing semi colons, etc; no code change

- Use an imperative, concise subject line after the prefix. Example:

```
feat: add validation to user create endpoint

Fixes: #123
```

- Keep commits small and logically grouped.

## Code review

- Be responsive to review comments and update your PR accordingly.
- Add reviewers and request reviews from maintainers when ready.
- Squash or tidy commits if requested by reviewers.

## Security

If you discover a security vulnerability, please do not open a public issue. Instead, contact the maintainers privately (see README for contact details) so we can address it responsibly.

## Questions

If you're unsure where to start, check `README.md` or open an issue asking for guidance. Thank you for helping improve the project!
