<script lang="ts">
    import { untrack } from 'svelte';
    import { token, permissions } from '../store';
    import { showError } from '../toast';
    import { type User, listUsers, PERM_LIST_USERS } from '../user';

    let users = $state<User[]>([]);
    let loading = $state(true);
    let error = $state('');

    async function loadUsers(authToken: string) {
        if (!authToken) {
            users = [];
            loading = false;
            return;
        }
        loading = true;
        error = '';
        try {
            const data = await listUsers(authToken);
            if (authToken !== $token) {
                return;
            }
            users = data;
        } catch (e) {
            if (authToken !== $token) {
                return;
            }
            error = e instanceof Error ? e.message : String(e);
            showError(e);
        } finally {
            if (authToken === $token) {
                loading = false;
            }
        }
    }

    // Permissions arrive with the session, after the token, so wait for them
    // instead of requesting a list the server would refuse.
    $effect(() => {
        const currentToken = $token;
        const allowed = $permissions.includes(PERM_LIST_USERS);
        untrack(() => {
            if (allowed) {
                loadUsers(currentToken);
            }
        });
    });
</script>

<h2 class="userList__title">Users</h2>
{#if !$permissions.includes(PERM_LIST_USERS)}
    <p class="userList__message">You do not have permission to list users.</p>
{:else if error}
    <p class="userList__message">{error}</p>
{:else if loading}
    <p class="userList__message">Loading...</p>
{:else if users.length === 0}
    <p class="userList__message">No users yet.</p>
{:else}
    <ul class="userList__items">
        {#each users as user, index (user.id)}
            <li class="userList__item">
                {index + 1}.
                <span>{user.email}</span>
                <span class="userList__id">{user.id}</span>
                {#each user.roles ?? [] as role (role)}
                    <span class="userList__role">{role}</span>
                {/each}
            </li>
        {/each}
    </ul>
{/if}

<style>
    .userList__title,
    .userList__message {
        text-align: center;
    }

    .userList__items {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 15px;
    }

    .userList__item {
        display: flex;
        gap: 10px;
        align-items: baseline;
        font-size: 20px;
    }

    .userList__id {
        font-family: monospace;
        font-size: 14px;
        font-style: italic;
    }

    .userList__role {
        font-size: 14px;
        font-weight: bold;
    }
</style>
