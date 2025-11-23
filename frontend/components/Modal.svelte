<script lang="ts">
    import { debounce } from '../util';
    import { maxNumberOfMessages, env } from '../store';

    let currentNumberOfMessages: number = $maxNumberOfMessages;
    let dialog: HTMLDialogElement | null = null;

    export function showModal() {
        if (!dialog) return;
        if (!dialog.open) {
            dialog.showModal();
        }
        currentNumberOfMessages = $maxNumberOfMessages;
    }

    function closeModal() {
        if (dialog?.open) {
            dialog.close();
        }
    }

    const commitMaxMessages = () => {
        if (currentNumberOfMessages > 0) {
            maxNumberOfMessages.set(currentNumberOfMessages);
        }
    };

    const debouncedCommit = debounce(commitMaxMessages, 1000);

    function handleInputChange(event: Event) {
        const target = event.target as HTMLInputElement;
        currentNumberOfMessages = Number(target.value);
        debouncedCommit();
    }

    function updateAndCloseModal() {
        commitMaxMessages();
        closeModal();
    }
</script>

<dialog bind:this={dialog} on:click|self={updateAndCloseModal}>
    <div>
        <p>Max number of messages</p>
        <input
            id="messages-input-box"
            type="number"
            min="0"
            bind:value={currentNumberOfMessages}
            on:input={handleInputChange}
            class:error-state={$maxNumberOfMessages <= 0}
        />
        <p>Model</p>
        <label>
            <input type="radio" bind:group={$env} name="currentEnv" id="dev" value="dev" />
            Development
        </label>
        <label>
            <input type="radio" bind:group={$env} name="currentEnv" id="prod" value="prod" />
            Production
        </label>
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
