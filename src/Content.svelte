<script lang="ts">
    import { hexy } from "hexy";
    export let content:messageType;

    function render(payload:string) {
        return hexy(atob(payload),{});
    }
</script>

<div class="column" id="content">
    {#if content}
        {content.srcHost}:{content.srcPort} -> {content.dstPort}<br />
        {#if content.rule}
            Rule: {content.rule}<br /><br />
        {/if}
        {#if content.payload}
            Payload:<br />
            <pre>{render(content.payload)}</pre>
        {/if}
    {/if}
</div>

<style>
    .column {
        flex: 50%;
        padding: 15px 20px;
    }
</style>
