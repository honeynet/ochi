# Agent instructions

## Before every commit

Run `npm run generate` and include any files it changes in the commit.

CI runs `npm run generate-check`, which regenerates `frontend/generated/chevrotain_dts.ts` and `public/syntax_diagram.html` and then runs prettier over all `*.svelte` and `*.ts` files. It fails if that leaves any diff, so unformatted files break the build.
