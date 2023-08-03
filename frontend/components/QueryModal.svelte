<script lang="ts">
    import { debounce } from '../util';
    import { filterActive, token } from '../store';
    import { escape } from 'svelte/internal';
    import { API_ENDPOINT } from '../constants';

    let dialog;
    let currentFilter: string;
    let currentDescription: string;
    let currentActive: boolean;

    export function showModal(filter: string) {
        if (!dialog.open) {
            dialog.showModal();
        }
        currentFilter = filter;
        currentActive = $filterActive;
    }

    function closeModal() {
        if (dialog.open) {
            dialog.close();
        }
    }

    function handleInputChange() {}

    async function saveQuery() {
        console.log('saving query');
        const res = await fetch(`${API_ENDPOINT}/queries`, {
            method: 'POST',
            headers: {
                Authorization: `Bearer ${$token}`,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                content: currentFilter,
                description: currentDescription,
                active: currentActive,
            }),
        });

        if (res.ok) {
            console.log('received success ' + res.text());
        } else {
            console.log('failed to save ' + res.text());
        }
    }

    async function confirmAndCloseModal() {
        console.log('closing modal');
        await saveQuery();
        closeModal();
    }
</script>

<dialog bind:this={dialog}>
    <div on:click|stopPropagation>
        <p>Content</p>
        <input id="messages-input-box" type="text" bind:value={currentFilter} />
        <p>Description</p>
        <input id="messages-input-box" type="text" bind:value={currentDescription} />
        <p>Active</p>
        <input id="messages-input-box" type="checkbox" bind:checked={currentActive} />
        <!-- <p>Model</p>
        <label>
            <input type="radio" bind:group={$env} name="currentEnv" id="dev" value="dev" />
            Development
        </label>
        <label>
            <input type="radio" bind:group={$env} name="currentEnv" id="prod" value="prod" />
            Production
        </label> -->
        <button on:click={confirmAndCloseModal}>Confirm</button>
        <button on:click={closeModal}>Cancel</button>
    </div>
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
    dialog > div {
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

    .error-state {
        border: 2px red solid;
        outline: none;
    }
</style>
