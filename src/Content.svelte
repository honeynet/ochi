<script lang="ts">
  import { hexy } from "hexy";
  export let content: messageType;

  function render(payload: string) {
    return hexy(atob(payload), {});
  }
</script>

<div class="column" id="content">
  {#if content}
    {content.srcHost}:{content.srcPort} -> {content.dstPort}<br />
    {#if content.rule}
      {content.rule}<br />
    {/if}
    {#if content.scanner}
      Scanner: {content.scanner}<br /><br />
    {/if}
    {#if content.payload}
      Payload:<br />
      <pre>{render(content.payload)}</pre>
      <a
        href={"data:text/json;charset=utf-8," +
          encodeURIComponent(JSON.stringify(content))}
        download="event.json">Download</a
      >
    {/if}
  {/if}
</div>

<style>
  .column {
    flex: 50%;
    padding: 15px 20px;
  }
</style>
