<script>
    import { createEventDispatcher } from 'svelte';

    const dispatch = createEventDispatcher();

    export let maxNumberOfMessages;
    export let env;
    let currentNumberOfMessages, currentEnv;
    let timeoutId;

    let dialog;

    export function showModal() {
        if (!dialog.open) {
            dialog.showModal();
        }
        currentNumberOfMessages = maxNumberOfMessages;
        // currentEnv = env;
    }

    function closeModal() {
        if (dialog.open) {
            dialog.close();
        }
    }

    function applyConfig() {
        // Let parent know about config update
        dispatch('configChange');
        maxNumberOfMessages = currentNumberOfMessages;
        env = currentEnv;
        dialog.close();
    }

    function handleInputChange() {
        clearTimeout(timeoutId);
        timeoutId = setTimeout(() => {
            if (currentNumberOfMessages > 0) {
                maxNumberOfMessages = currentNumberOfMessages;
                dispatch('configChange');
            }
        }, 2000);
    }

    function updateAndCloseModal() {
        if (currentNumberOfMessages > 0) {
            maxNumberOfMessages = currentNumberOfMessages;
            dispatch('configChange');
        }
        closeModal();
    }
</script>

<dialog bind:this={dialog} on:click|self={updateAndCloseModal}>
    <div on:click|stopPropagation>
        <p>Max number of messages</p>
        <input
            id="messages-input-box"
            type="number"
            min="0"
            bind:value={currentNumberOfMessages}
            on:input={handleInputChange}
            class:error-state={maxNumberOfMessages <= 0}
        />
        <p>Model</p>
        <label>
            <input type="radio" bind:group={env} name="currentEnv" id="dev" value="dev" />
            Development
        </label>
        <label>
            <input type="radio" bind:group={env} name="currentEnv" id="prod" value="prod" />
            Production
        </label>
        <!-- <button disabled={currentNumberOfMessages < 0} on:click={applyConfig}>Apply</button> -->
        <button on:click={closeModal}>Close</button>
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
