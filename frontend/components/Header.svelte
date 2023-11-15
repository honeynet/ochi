<script lang="ts">
    import { url } from '@roxi/routify';
    import SSOButton from './SSOButton.svelte';
    import LogoutButton from './LogoutButton.svelte';
    import SSORevokeButton from './SSORevokeButton.svelte';
    import { isAuthenticated } from '../store';
    export let path: string;
    export let pathText: string;
    import { onDestroy } from 'svelte';

    // subscribe to the authentication status
    let isLoggedIn: boolean;
    const isAuthenticatedUnsubscribe = isAuthenticated.subscribe((status) => {
        isLoggedIn = status;
    });

    onDestroy(() => {
        isAuthenticatedUnsubscribe();
    });
</script>

<header class="header">
    <a class="header__link" target="_blank" href="https://github.com/honeynet/ochi">Ochi</a>
    <div class="header__container">
        {#if !isLoggedIn}
            <SSOButton />
        {:else}
            <a class="header__link" href={$url(path)}>{pathText}</a>
            <LogoutButton />
            <SSORevokeButton />
        {/if}
    </div>
</header>

<style>
    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        border-bottom-style: solid;
        padding-bottom: 10px;
        border-width: 1px;
        margin-right: 20px;
        margin-left: 20px;
    }

    .header__link {
        font-size: 20px;
    }

    .header__container {
        display: flex;
        gap: 20px;
        align-items: center;
    }
</style>
