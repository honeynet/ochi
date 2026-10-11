# Contributing to Ochi

Thanks for considering a contribution! Ochi is the live event UI for the [Glutton](https://github.com/mushorg/glutton) honeypot, so small, source-verified changes are easier to review than broad rewrites. Bug fixes, features, documentation, and ideas are all welcome, and so is reviewing and testing open pull requests.

For the Go and Node.js toolchain, local setup, and how to run Ochi (including feeding it events from a local Glutton instance), see the [README](README.md). This file only covers how to contribute.

## Workflow

1. **Pick an issue** from the [tracker](https://github.com/honeynet/ochi/issues). Useful labels: `good first issue`, `help wanted`, `enhancement`, `go`, `javascript`. If nothing fits, open a new issue describing the bug or proposal. Questions about setup or scope go to [Discussions](https://github.com/honeynet/ochi/discussions).
2. **Comment before starting** so a maintainer can confirm scope or redirect, and to avoid duplicate work.
3. **Fork the repo**, clone your fork, and add the upstream remote:
   ```bash
   git remote add upstream https://github.com/honeynet/ochi.git
   ```
4. **Create a branch:**
   ```bash
   git fetch upstream
   git checkout -b <type>/<short-description> upstream/main
   ```
5. **Make changes and test**, keeping commits small and focused (see [Guidelines](#guidelines)).
6. **Open a PR** from your fork against `honeynet/ochi:main`.

## Guidelines

- **Keep PRs small and source-verified.** Split large work into a thin first slice, and keep unrelated cleanup in its own PR.
- **Add tests** beside the code you change: Go tests next to the package in `backend/`, Jest tests in `frontend/__tests__/`, and Playwright specs in `e2e/` for navigation or page-level behavior.
- **Update docs in the same PR** when behavior changes:
  - Build, CI, Docker, or Go/Node version → [README](README.md)
  - Config defaults → [README](README.md) and [config.yaml](config.yaml) / [config.docker.yaml](config.docker.yaml)
  - Filter/query DSL syntax → [frontend/syntax.md](frontend/syntax.md)
- **Format:** run `gofmt` on Go code. Frontend `*.svelte` and `*.ts` files are formatted with prettier via `npm run format`.
- **Regenerate after frontend changes:** if you touch any `*.svelte` or `*.ts` file, the DSL grammar (`frontend/dsl.ts`), or `scripts/generate.ts`, run `npm run generate` and commit the resulting changes. CI runs `npm run generate-check` and fails if `frontend/generated/chevrotain_dts.ts`, `public/syntax_diagram.html`, or formatting are out of date.
- **Test before pushing:** run `go test ./...` for the backend, and `npm run check`, `npm run test`, and `npm run test:e2e` for the frontend. These mirror what CI runs.
- **Don't commit temporary local changes.** For example, revert any edit to the default in `frontend/store.ts` before opening a PR.
- **Write a clear PR description:** what changed, how you tested it, and which docs moved with it. Link the issue (e.g., `Fixes #<issue-number>`).

## Reporting bugs

When opening a bug report, please include:
- What you expected to happen and what actually happened.
- Steps to reproduce.
- Your OS, Go version, and Node.js version.
- Relevant logs or screenshots.

## AI agents

AI coding agents (Claude Code, Copilot, Cursor, Codex, and similar) are welcome, but their use **must be disclosed**.

- **Disclose in the PR description** which tool you used and for what (e.g. "component and tests drafted with Claude Code, reviewed and edited by hand"). Commits written largely by an agent should also carry a `Co-Authored-By:` trailer naming it. Issues and review comments drafted by an agent need the same note.
- **You own the change.** Read every line before you open the PR, run the tests yourself, and be ready to explain and defend it in review. Undisclosed or unreviewed agent output may be closed without review.
- **Verify against the source.** Agents invent field names and APIs. Check event fields against what Glutton actually produces, and Svelte/Routify usage against the versions pinned in [package.json](package.json).
- **Point your agent at [AGENTS.md](AGENTS.md).** It describes the steps agents must follow in this repo.
- **Keep secrets and private data out of prompts.** Don't paste credentials, tokens from your `config.yaml`, real attacker IPs beyond what an issue already shows, or non-public event data into third-party tools.

## License

By contributing, you agree that your contributions will be licensed under the project's [GPL-3.0 license](LICENSE).