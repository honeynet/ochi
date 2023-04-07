<script lang="ts">
    export let showModal;

    let messages: messageType[] = [];

    let noOfMessages: number;
    let dialog;
    let inputMessages: number;
    let isDevOn: boolean;
    let conn: WebSocket;
    function defineMessages(event: any) {
        let chosenValue = event.target[0].value;
        let presentMessages = messages.length;

        if (chosenValue <= 0) {
            return;
        }

        if (chosenValue < presentMessages) {
            messages = messages.slice(messages.length - chosenValue, messages.length);
        }

        noOfMessages = chosenValue;
    }

    function toggleMode(event: any) {
        let chosenMode = event.target.id;

        if (chosenMode == 'dev') {
            isDevOn = true;
        } else if (chosenMode == 'prod') {
            isDevOn = false;
            if (conn != null) {
                conn.close();
            }
        }
    }

    $: if (dialog && showModal) dialog.showModal();
</script>

<dialog
    bind:this={dialog}
    on:close={() => (showModal = false)}
    on:click|self={() => dialog.close()}
>
    <div on:click|stopPropagation>
        <form id="configmodal" on:submit|preventDefault={defineMessages}>
            <p>Number of messages</p>
            <input
                id="messages-input-box"
                type="number"
                min="0"
                bind:value={inputMessages}
                class:error-state={inputMessages <= 0}
            />
            <p>Mode</p>
            <label>
                <input type="radio" name="radio-group" id="dev" on:click={toggleMode} />
                Development
            </label>
            <label>
                <input type="radio" name="radio-group" checked id="prod" on:click={toggleMode} />
                Production
            </label>
            <button disabled={inputMessages < 0} type="submit">Apply</button>
        </form>
        <slot />
        <button on:click={() => dialog.close()}>Close</button>
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
</style>
