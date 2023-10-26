<script lang="ts">
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

<p on:click={click} on:keypress={click} bind:this={element}>
    {message.sensorID} | {message.srcHost}:{message.srcPort} -> {message.dstPort}:
    {#if message.handler}{message.handler}{:else}{message.rule}{/if}
    {#if message.scanner}"{message.scanner}"{/if}
    <u>Details</u>
</p>

<style>
    p {
        margin: 5px 0 0 0;
        font-family: monospace;
    }
</style>
