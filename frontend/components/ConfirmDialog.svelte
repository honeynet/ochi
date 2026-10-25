<script lang="ts">
    import type { Snippet } from 'svelte';

    interface Props {
        confirmText?: string;
        cancelText?: string;
        busy?: boolean;
        onconfirm: () => void | Promise<void>;
        oncancel?: () => void;
        onclose?: () => void;
        children?: Snippet;
    }

    let {
        confirmText = 'Delete',
        cancelText = 'Cancel',
        busy = false,
        onconfirm,
        oncancel,
        onclose,
        children,
    }: Props = $props();

    let dialog = $state<HTMLDialogElement | null>(null);

    export function showModal() {
        if (!dialog) return;
        if (!dialog.open) {
            dialog.showModal();
        }
    }

    export function close() {
        if (dialog?.open) {
            dialog.close();
        }
    }
</script>

<dialog
    bind:this={dialog}
    oncancel={(e) => {
        if (busy) {
            e.preventDefault();
            return;
        }
        oncancel?.();
    }}
    onclose={() => {
        onclose?.();
    }}
    onclick={(event) => {
        if (busy) return;
        if (event.target === event.currentTarget) {
            close();
            oncancel?.();
        }
    }}
>
    <div class="confirm-dialog">
        {@render children?.()}
        <div class="confirm-buttons">
            <button
                onclick={() => {
                    close();
                    oncancel?.();
                }}
                disabled={busy}
            >
                {cancelText}
            </button>
            <button onclick={onconfirm} disabled={busy}>
                {confirmText}
            </button>
        </div>
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

    .confirm-dialog {
        padding: 1em;
    }

    .confirm-buttons {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
    }
</style>
