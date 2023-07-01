<script lang="ts">
    import SSOButton from './SSOButton.svelte';
    import LogoutButton from './LogoutButton.svelte';
    import SSORevokeButton from './SSORevokeButton.svelte';
    export let isLoggedIn: boolean;
</script>

<header class="header">
    <a class="header__link" target="_blank" href="https://github.com/honeynet/ochi">Ochi</a>

    {#if !isLoggedIn}
        <SSOButton />
    {:else}
        <LogoutButton />
        <SSORevokeButton />
    {/if}
</header>

<style>
    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        border-bottom-style: solid;
        border-width: 1px;
    }

    .header__link {
        font-size: 20px;
    }
</style>
