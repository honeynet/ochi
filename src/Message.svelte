<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";

  const dispatch = createEventDispatcher();

  export let message: messageType;
  export let follow: boolean;
  let element: HTMLElement;

  function click() {
    dispatch("message", message);
  }

  onMount(() => {
    follow && element.scrollIntoView();
  });
</script>

<p on:click={click} bind:this={element}>
  {message.srcHost}:{message.srcPort} -> {message.dstPort}: {message.rule}
  {#if message.scanner}"{message.scanner}"{/if}
  {#if message.payload}: <u>Payload</u>{/if}
</p>

<style>
  p {
    margin: 5px 0 0 0;
  }
</style>
