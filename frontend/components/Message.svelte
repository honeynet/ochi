<script lang="ts">
    import { onMount } from 'svelte';
    import type { Event } from '../event';
    import { currentEvent } from '../store';
    export let message: Event;
    export let follow: boolean;
    let element: HTMLButtonElement | null = null;

    function click() {
        currentEvent.set(message);
    }

    onMount(() => {
        if (follow) {
            element?.scrollIntoView();
        }
    });
</script>

<button type="button" class="message" on:click={click} bind:this={element}>
    {message.sensorID} | {message.srcHost}:{message.srcPort} -> {message.dstPort}:
    {#if message.handler}{message.handler}{:else}{message.rule}{/if}
    {#if message.scanner}"{message.scanner}"{/if}
    <u>Details</u>
</button>

<style>
    .message {
        margin: 5px 0 0 0;
        font-family: monospace;
        background: none;
        border: none;
        padding: 0;
        text-align: left;
        cursor: pointer;
    }
</style>
