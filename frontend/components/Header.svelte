<script lang="ts">
    import { url, isActive } from '@roxi/routify';
    import SSOButton from './SSOButton.svelte';
    import LogoutButton from './LogoutButton.svelte';
    import SSORevokeButton from './SSORevokeButton.svelte';
    import Config from './Config.svelte';
    import Filter from './Filter.svelte';
    import { isAuthenticated, permissions } from '../store';
    import { PERM_LIST_USERS } from '../user';

    interface NavLink {
        path: string;
        label: string;
        perm?: string;
    }

    const navLinks: NavLink[] = [
        { path: '/index', label: 'Live' },
        { path: '/myqueries', label: 'Queries' },
        { path: '/myevents', label: 'Events' },
        { path: '/mysensors', label: 'Sensors' },
        { path: '/users', label: 'Users', perm: PERM_LIST_USERS },
    ];
</script>

<header class="header">
    <div class="header__left">
        <a class="header__link header__brand" href={$url('/')}>Ochi</a>
        {#if $isAuthenticated}
            <nav class="header__nav" aria-label="Main navigation">
                {#each navLinks as link}
                    {#if !link.perm || $permissions.includes(link.perm)}
                        <a
                            class="header__link"
                            class:header__link--active={$isActive(link.path)}
                            aria-current={$isActive(link.path) ? 'page' : undefined}
                            href={$url(link.path === '/index' ? '/' : link.path)}
                        >
                            {link.label}
                        </a>
                    {/if}
                {/each}
            </nav>
        {:else if !$isActive('/index')}
            <a class="header__link" href={$url('/')}>Go back</a>
        {/if}
    </div>
    <div class="header__center" hidden={!$isActive('/index')}>
        <Filter />
    </div>
    <div class="header__right">
        {#if !$isAuthenticated}
            <SSOButton />
        {:else}
            <LogoutButton />
            <SSORevokeButton />
        {/if}
        <a
            class="header__link"
            target="_blank"
            rel="noreferrer"
            href="https://github.com/honeynet/ochi">GitHub</a
        >
        <Config />
    </div>
</header>

<style>
    .header {
        display: grid;
        grid-template-columns: auto 1fr auto;
        align-items: center;
        border-bottom-style: solid;
        padding-top: 10px;
        padding-bottom: 10px;
        border-width: 1px;
        margin-right: 20px;
        margin-left: 20px;
        gap: 16px;
    }

    .header :global(input),
    .header :global(button) {
        margin: 0;
    }

    .header__left {
        display: flex;
        align-items: center;
        gap: 16px;
        justify-self: start;
        flex-wrap: wrap;
    }

    .header__nav {
        display: flex;
        align-items: center;
        gap: 16px;
        flex-wrap: wrap;
    }

    .header__center {
        justify-self: center;
        width: 100%;
        max-width: 600px;
        min-width: 0;
    }

    .header__center[hidden] {
        display: none;
    }

    .header__right {
        display: flex;
        gap: 16px;
        align-items: center;
        justify-self: end;
        flex-wrap: wrap;
    }

    .header__link {
        font-size: 20px;
    }

    .header__link--active {
        font-weight: 600;
        text-decoration: underline;
    }

    .header__brand {
        font-weight: 600;
    }
</style>
