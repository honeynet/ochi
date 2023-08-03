<script lang="ts">
    export let query: Object;

    import { onMount } from 'svelte';
    import type { Event } from '../event';
    import { currentEvent } from '../store';
    export let message: Event;
    export let follow: boolean;
    let element: HTMLElement;

    function click() {
        currentEvent.set(message);
    }

    onMount(() => {
        follow && element.scrollIntoView();
    });
</script>

<p on:click={click} bind:this={element}>
    {message.sensorID.split('-')[0]} | {message.srcHost}:{message.srcPort} -> {message.dstPort}:
    {#if message.handler}{message.handler}{:else}{message.rule}{/if}
    {#if message.scanner}"{message.scanner}"{/if}
    {#if message.payload}: <u>Payload</u>{/if}
</p>

<style>
    p {
        margin: 5px 0 0 0;
        font-family: monospace;
    }
</style>
