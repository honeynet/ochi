<script lang="ts">
    import { onMount } from 'svelte';
    import type { Event } from '../event';
    import { currentEvent } from '../store';

    interface Props {
        message: Event;
        follow: boolean;
    }

    let { message, follow }: Props = $props();
    let element = $state<HTMLButtonElement | null>(null);

    function click() {
        currentEvent.set(message);
    }

    onMount(() => {
        if (follow) {
            element?.scrollIntoView();
        }
    });
</script>

<button type="button" class="message" onclick={click} bind:this={element}>
    {message.sensorID} | {message.srcHost}:{message.srcPort} -> {message.dstPort}:
    {#if message.handler}{message.handler}{:else}{message.rule}{/if}
    {#if message.scanner}"{message.scanner}"{/if}
    <u>Details</u>
</button>

<style>
    .message {
        display: block;
        width: 100%;
        margin: 5px 0 0 0;
        font-family: monospace;
        background: none;
        border: none;
        padding: 0;
        text-align: left;
        cursor: pointer;
    }
</style>
