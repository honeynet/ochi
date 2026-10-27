<svelte:options runes={true} />

<script lang="ts">
    import { toasts, dismissToast, pauseToast, resumeToast } from '../toast';
</script>

<div class="toasts" aria-live="assertive">
    {#each $toasts as toast (toast.id)}
        <div
            class="toast toast--{toast.type}"
            role={toast.type === 'error' ? 'alert' : 'status'}
            onmouseenter={() => pauseToast(toast.id)}
            onmouseleave={() => resumeToast(toast.id)}
            onfocusin={() => pauseToast(toast.id)}
            onfocusout={() => resumeToast(toast.id)}
        >
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
        border-radius: 4px;
        font-family: monospace;
        font-size: 12px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
        background: #fdecea;
        color: #8a1f17;
        border: 1px solid #f5c6c2;
    }

    .toast--success {
        background: #e6f4ea;
        color: #137333;
        border: 1px solid #ceead6;
    }

    .toast--info {
        background: #e8f0fe;
        color: #174ea6;
        border: 1px solid #d2e3fc;
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
