<script lang="ts">
    import { hexy } from 'hexy';
    import type { Event } from './event';
    export let content: Event;

    function render(payload: string) {
        return hexy(atob(payload), {});
    }
</script>

<div class="column" id="content">
    {#if content}
        {content.srcHost}:{content.srcPort} -> {content.dstPort}<br />
        {#if content.handler}
            Handler: {content.handler}<br />
        {/if}
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
                href={'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(content))}
                download="event.json">Download</a
            >
        {/if}
        {#if content.decoded}
            <br /><br />
            <pre>{JSON.stringify(content.decoded, null, 2)}</pre>
        {/if}
    {/if}
</div>

<style>
    .column {
        flex: 50%;
        padding: 15px 20px;
    }
</style>
