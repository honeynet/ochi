# Contributing to Ochi

Thanks for considering a contribution! Ochi is the live event UI for the [Glutton](https://github.com/mushorg/glutton) honeypot, so small, source-verified changes are easier to review than broad rewrites. Contributions of all sizes are welcome, including bug fixes, features, documentation, and ideas.

See the [README](README.md) for the Go and Node.js toolchain, local setup, and how to feed Ochi events from a local Glutton instance.

## Ways to contribute

- Report a bug or suggest an improvement by [opening an issue](https://github.com/honeynet/ochi/issues).
- Improve documentation (setup steps, comments, examples).
- Pick up an existing issue, especially those labeled `good first issue` or `help wanted`.
- Review open pull requests and test them locally.

## Getting started & Workflow

1. **Fork and clone:**
   ```bash
   git clone https://github.com/<your-username>/ochi.git
   cd ochi
   git remote add upstream https://github.com/honeynet/ochi.git
   ```
2. **Setup environment:** Follow the [README](README.md) to install dependencies and run the project. For frontend-only work, you can toggle to simulated events directly from the **Config** button in the UI.
3. **Pick an issue:** Pick an issue from the [tracker](https://github.com/honeynet/ochi/issues). Useful labels: `good first issue`, `help wanted`, `enhancement`, `go`, `javascript`. Questions about setup or scope go to [Discussions](https://github.com/honeynet/ochi/discussions).
4. **Comment before starting:** Comment on the issue so a maintainer can confirm scope or redirect and avoid duplicate work.
5. **Create a branch:**
   ```bash
   git fetch upstream
   git checkout -b <type>/<short-description> upstream/main
   ```
6. **Make changes & test:** Keep commits small and focused. Run tests and formatting before pushing.
7. **Open a PR:** Push to your fork and open a pull request against `honeynet/ochi:main`.

## Guidelines

- **Keep PRs small and source-verified.** Split large work into a thin first slice.
- **Add tests** beside the code you change: Go tests next to the package in `backend/`, Jest tests in `frontend/__tests__/`, and Playwright specs in `e2e/` for navigation or page-level behavior.
- **Update docs in the same PR** when behavior changes:
  - Build, CI, Docker, or Go/Node version → [README](README.md)
  - Config defaults → [README](README.md) and [config.yaml](config.yaml) / [config.docker.yaml](config.docker.yaml)
  - Filter/query DSL syntax → [frontend/syntax.md](frontend/syntax.md)

## Code style and PRs

- **Format:** run `gofmt` on Go code. Frontend `*.svelte` and `*.ts` files are formatted with prettier via `npm run format`.
- **Regenerate after frontend changes:** if you touch any `*.svelte` or `*.ts` file, the DSL grammar (`frontend/dsl.ts`), or `scripts/generate.ts`, run `npm run generate` and commit the resulting changes. CI runs `npm run generate-check` and fails if `frontend/generated/chevrotain_dts.ts`, `public/syntax_diagram.html`, or formatting are out of date.
- **Test before pushing:** run `go test ./...` for the backend, and `npm run check`, `npm run test`, and `npm run test:e2e` for the frontend. These mirror what CI runs.
- **No local test hacks:** use the **Config → Development** toggle in the UI for simulated events instead of modifying code. If you temporarily changed the default in `frontend/store.ts`, revert it before opening a PR.
- **Write a focused PR:** describe what changed, how you tested it, and which docs moved with it. Keep unrelated cleanup in its own PR.
- **Link the issue:** reference the issue in the PR description (e.g., `Fixes #176`).

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
