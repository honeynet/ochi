<script lang="ts">
    import { token, userQueries } from '../store';
    import { createQuery, getQueries, updateQuery } from '../query';
    import type { Query } from '../query';

    let dialog: HTMLDialogElement | null = null;
    let queryToEdit: Query = {};

    export function showModal(objectToEdit: Query) {
        if (!dialog) return;
        if (!dialog.open) {
            dialog.showModal();
        }
        queryToEdit = objectToEdit;
    }

    function closeModal() {
        if (dialog?.open) {
            dialog.close();
        }
    }

    async function saveQueryAndReload() {
        console.log('saving query');
        await createQuery(
            {
                content: queryToEdit.content,
                description: queryToEdit.description,
                active: queryToEdit.active,
            },
            $token,
        );
        await reloadQueries();
    }

    async function reloadQueries() {
        const queries = await getQueries($token);
        userQueries.set(queries);
    }

    async function updateQueryAndReload() {
        await updateQuery(
            {
                id: queryToEdit.id,
                content: queryToEdit.content,
                description: queryToEdit.description,
                active: queryToEdit.active,
            },
            $token,
        );
        queryToEdit = {};
        await reloadQueries();
    }

    async function confirmAndCloseModal() {
        console.log('closing modal');
        await saveQueryAndReload();
        closeModal();
    }

    async function updateAndCloseModal() {
        await updateQueryAndReload();
        closeModal();
    }
</script>

<dialog bind:this={dialog}>
    <form class="queryModal__form">
        <label
            >Content
            <input id="messages-input-box" type="text" bind:value={queryToEdit.content} />
        </label>
        <label
            >Description
            <input id="messages-input-box" type="text" bind:value={queryToEdit.description} />
        </label>
        <!-- <label
            >Active
            <input id="messages-input-box" type="checkbox" bind:checked={queryToEdit.active} />
        </label> -->
        {#if queryToEdit && queryToEdit.id}
            <button type="button" on:click={updateAndCloseModal}>Update</button>
        {:else}
            <button type="button" on:click={confirmAndCloseModal}>Confirm</button>
        {/if}

        <button type="button" on:click={closeModal}>Cancel</button>
    </form>
</dialog>

<style>
    dialog {
        max-width: 32em;
        border-radius: 0.2em;
        border: none;
        padding: 0;
    }
    dialog::backdrop {
        background: rgba(0, 0, 0, 0.3);
    }
    dialog > form {
        padding: 1em;
    }
    dialog[open] {
        animation: zoom 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    dialog[open]::backdrop {
        animation: fade 0.2s ease-out;
    }
    button {
        display: block;
    }

    .queryModal__form {
        margin: 20px 20px;
    }
</style>
