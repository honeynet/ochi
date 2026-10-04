<script lang="ts">
    import type { Snippet } from 'svelte';
    import { url } from '@roxi/routify';
    import SSOButton from './SSOButton.svelte';
    import LogoutButton from './LogoutButton.svelte';
    import SSORevokeButton from './SSORevokeButton.svelte';
    import Config from './Config.svelte';
    import { isAuthenticated } from '../store';

    interface Props {
        path: string;
        pathText: string;
        headerCenter?: Snippet;
    }

    let { path, pathText, headerCenter }: Props = $props();
</script>

<header class="header">
    <div class="header__left">
        <a class="header__link" target="_blank" href="https://github.com/honeynet/ochi">Ochi</a>
    </div>
    <div class="header__center">
        {@render headerCenter?.()}
    </div>
    <div class="header__right">
        {#if !$isAuthenticated}
            <SSOButton />
        {:else}
            <a class="header__link" href={$url(path)}>{pathText}</a>
            <LogoutButton />
            <SSORevokeButton />
        {/if}
        <Config />
    </div>
</header>

<style>
    .header {
        display: grid;
        grid-template-columns: 1fr auto 1fr;
        align-items: center;
        border-bottom-style: solid;
        padding-top: 10px;
        padding-bottom: 10px;
        border-width: 1px;
        margin-right: 20px;
        margin-left: 20px;
        gap: 12px;
    }

    .header :global(input),
    .header :global(button) {
        margin: 0;
    }

    .header__left {
        justify-self: start;
    }

    .header__center {
        justify-self: center;
        min-width: 0;
    }

    .header__right {
        display: flex;
        gap: 20px;
        align-items: center;
        justify-self: end;
    }

    .header__link {
        font-size: 20px;
    }
</style>
