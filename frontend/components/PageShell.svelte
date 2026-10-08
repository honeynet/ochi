<script lang="ts">
    import type { Snippet } from 'svelte';
    import Header from './Header.svelte';

    interface Props {
        path: string;
        pathText: string;
        headerCenter?: Snippet;
        showEvents?: boolean;
        children: Snippet;
    }

    let { path, pathText, headerCenter, showEvents, children }: Props = $props();
</script>

<div class="shell">
    <Header {path} {pathText} {headerCenter} {showEvents} />
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
    }
</style>
