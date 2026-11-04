<script lang="ts">
    import type { Snippet } from 'svelte';
    import Header from './Header.svelte';

    interface Props {
        children: Snippet;
    }

    let { children }: Props = $props();
</script>

<div class="shell">
    <Header />
    <div class="shell__body">
        {@render children()}
    </div>
</div>

<style>
    .shell {
        display: flex;
        flex-direction: column;
        height: 100%;
        min-height: 0;
    }

    .shell__body {
        display: flex;
        flex-direction: column;
        flex: 1;
        min-height: 0;
        overflow: auto;
        /* Keep the page's own layers (sticky table headers, overlays) below anything the
           header opens over the content, such as the filter suggestions. */
        isolation: isolate;
    }
</style>
