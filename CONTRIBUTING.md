# Contributing to Ochi

Thanks for considering a contribution. Ochi is the live event UI for the [Glutton](https://github.com/mushorg/glutton) honeypot, so small source-verified changes are easier to review than broad rewrites.

See the [README](README.md) for the Go and Node.js toolchain, local setup, and how to feed Ochi events from a local Glutton instance.

## Guidelines

- **Pick an issue** from the [tracker](https://github.com/honeynet/ochi/issues). Useful labels: `good first issue`, `help wanted`, `enhancement`, `go`, `javascript`. Reproduce older issues against `main` before starting. Questions about setup or scope go to [Discussions](https://github.com/honeynet/ochi/discussions).
- **Comment before you start** so a maintainer can confirm scope or redirect.
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
- **Revert local test hacks:** if you swapped `dial()` for `test()` in `frontend/App.svelte` to generate fake events, switch it back before opening a PR.
- **Write a focused PR:** describe what changed, how you tested it, and which docs moved with it. Keep unrelated cleanup in its own PR.

## AI agents

AI coding agents (Claude Code, Copilot, Cursor, Codex, and similar) are welcome, but their use **must be disclosed**.

- **Disclose in the PR description** which tool you used and for what (e.g. "component and tests drafted with Claude Code, reviewed and edited by hand"). Commits written largely by an agent should also carry a `Co-Authored-By:` trailer naming it. Issues and review comments drafted by an agent need the same note.
- **You own the change.** Read every line before you open the PR, run the tests yourself, and be ready to explain and defend it in review. Undisclosed or unreviewed agent output may be closed without review.
- **Verify against the source.** Agents invent field names and APIs. Check event fields against what Glutton actually produces, and Svelte/Routify usage against the versions pinned in [package.json](package.json).
- **Point your agent at [AGENTS.md](AGENTS.md).** It describes the steps agents must follow in this repo.
- **Keep secrets and private data out of prompts.** Don't paste credentials, tokens from your `config.yaml`, real attacker IPs beyond what an issue already shows, or non-public event data into third-party tools.
