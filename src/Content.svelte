<script lang="ts">
    import { hexy } from 'hexy';
    import type { Event } from './event';
    import { currentEvent } from './store';

    function render(payload: string) {
        return hexy(atob(payload), {});
    }
</script>

<div class="column" id="content">
    {#if $currentEvent}
        {$currentEvent.srcHost}:{$currentEvent.srcPort} -> {$currentEvent.dstPort}<br />
        {#if $currentEvent.handler}
            Handler: {$currentEvent.handler}<br />
        {/if}
        {#if $currentEvent.rule}
            {$currentEvent.rule}<br />
        {/if}
        {#if $currentEvent.scanner}
            Scanner: {$currentEvent.scanner}<br /><br />
        {/if}
        {#if $currentEvent.payload}
            Payload:<br />
            <pre>{render($currentEvent.payload)}</pre>
            <a
                href={'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify($currentEvent))}
                download="event.json">Download</a
            >
        {/if}
        {#if $currentEvent.decoded}
            <br /><br />
            <pre>{JSON.stringify($currentEvent.decoded, null, 2)}</pre>
        {/if}
    {/if}
</div>

<style>
    .column {
        flex: 50%;
        padding: 15px 20px;
    }
</style>
