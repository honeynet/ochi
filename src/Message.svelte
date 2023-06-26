<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import type { Event } from './event';

    const dispatch = createEventDispatcher();

    export let message: Event;
    export let follow: boolean;
    let element: HTMLElement;

    function click() {
        dispatch('message', message);
    }

    onMount(() => {
        follow && element.scrollIntoView();
    });
</script>

<p on:click={click} bind:this={element}>
    {message.srcHost}:{message.srcPort} -> {message.dstPort}:
    {#if message.handler}{message.handler}{:else}{message.rule}{/if}
    {#if message.scanner}"{message.scanner}"{/if}
    {#if message.payload}: <u>Payload</u>{/if}
</p>

<style>
    p {
        margin: 5px 0 0 0;
    }
</style>
