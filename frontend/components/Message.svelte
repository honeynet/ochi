<script lang="ts">
    import { onMount } from 'svelte';
    import type { Event } from '../event';
    import { displayRule } from '../event';
    import { currentEvent } from '../store';

    interface Props {
        message: Event;
        follow: boolean;
    }

    let { message, follow }: Props = $props();
    let element = $state<HTMLButtonElement | null>(null);

    let source = $derived(`${message.srcHost}:${message.srcPort}`);
    let destPort = $derived(`:${message.dstPort}`);
    let handlerLabel = $derived(message.handler || displayRule(message) || '');
    let selected = $derived($currentEvent === message);

    function click() {
        currentEvent.set(message);
    }

    onMount(() => {
        if (follow) {
            element?.scrollIntoView();
        }
    });
</script>

<button
    type="button"
    class={['message', { selected }]}
    onclick={click}
    bind:this={element}
    aria-pressed={selected}
>
    <span class="cell sensor" title={message.sensorID}>{message.sensorID}</span>
    <span class="cell source" title={message.srcPtr || source}>{source}</span>
    <span class="cell dest">{destPort}</span>
    <span class="cell handler" title={handlerLabel || undefined}>{handlerLabel}</span>
    <span class="cell scanner" title={message.scanner}>{message.scanner ?? ''}</span>
    <span class="cell end" title={message.endReason}>{message.endReason ?? ''}</span>
    <span class="cell frames">{message.frameCount ?? ''}</span>
    <span class="cell details">Details</span>
</button>

<style>
    .message {
        display: grid;
        grid-template-columns: var(--event-cols);
        column-gap: 6px;
        align-items: center;
        width: 100%;
        margin: 0;
        padding: 1px 0;
        font-family: monospace;
        font-size: 12px;
        line-height: 1.2;
        background: none;
        border: none;
        border-bottom: 1px solid #eee;
        text-align: left;
        cursor: pointer;
    }

    .message:hover,
    .selected {
        background: #f5f5f5;
    }

    .cell {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .frames {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    .details {
        text-decoration: underline;
    }
</style>
