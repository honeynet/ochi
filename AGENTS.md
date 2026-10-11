# Agent instructions

## Run `npm run generate` after changing frontend code

When you finish a change that touches any `*.svelte` or `*.ts` file, or the DSL grammar (`frontend/dsl.ts`) or `scripts/generate.ts`, run `npm run generate` before reporting the work as done. Do this whether or not you are committing. Leave any files it changes in the working tree, and include them in the commit if you commit.

CI runs `npm run generate-check`, which regenerates `frontend/generated/chevrotain_dts.ts` and `public/syntax_diagram.html` and then runs prettier over all `*.svelte` and `*.ts` files. It fails if that leaves any diff, so unformatted files or stale generated files break the build.
