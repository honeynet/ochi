<svelte:options runes={true} />

<script lang="ts">
    import { toasts, dismissToast } from '../toast';
</script>

<div class="toasts" aria-live="assertive">
    {#each $toasts as toast (toast.id)}
        <div class="toast" role="alert">
            <span class="toast__message">{toast.message}</span>
            <button
                class="toast__dismiss"
                aria-label="Dismiss"
                onclick={() => dismissToast(toast.id)}
            >
                &times;
            </button>
        </div>
    {/each}
</div>

<style>
    .toasts {
        position: fixed;
        top: 16px;
        right: 16px;
        z-index: 1000;
        display: flex;
        flex-direction: column;
        gap: 8px;
        max-width: 360px;
    }

    .toast {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        padding: 10px 12px;
        background: #fdecea;
        color: #8a1f17;
        border: 1px solid #f5c6c2;
        border-radius: 4px;
        font-family: monospace;
        font-size: 12px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
    }

    .toast__message {
        flex: 1;
        word-break: break-word;
    }

    .toast__dismiss {
        margin: 0;
        padding: 0 4px;
        background: none;
        border: none;
        color: inherit;
        font-size: 16px;
        line-height: 1;
        cursor: pointer;
    }
</style>
