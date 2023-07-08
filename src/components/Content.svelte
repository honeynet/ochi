<script lang="ts">
    import { hexy } from 'hexy';
    import { currentEvent } from '../store';

    function render(payload: string) {
        const result = hexy(atob(payload), { width: 16 });
        const resultLines = result.split('\n');
        let addressStr = '';
        let hexStr = '';
        let plainStr = '';
        resultLines.forEach((item, idx) => {
            if (item) {
                addressStr += (idx > 0 ? '\n' : '') + item.substring(0, item.indexOf(':') + 1);
                hexStr += (idx > 0 ? '\n' : '') + item.substring(item.indexOf(':') + 2, 51);
                plainStr += (idx > 0 ? '\n' : '') + item.substring(51);
            }
        });
        return [
            { name: 'addressStr', content: addressStr },
            { name: 'hexStr', content: hexStr },
            { name: 'plainStr', content: plainStr },
        ];
    }

    let renderResults;
    $: if ($currentEvent) {
        renderResults = render($currentEvent.payload);
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
            <div class="pre">
                {#each renderResults as renderResult}
                    <div class={renderResult.name}>
                        {renderResult.content}
                    </div>
                {/each}
            </div>
            <a
                href={'data:text/json;charset=utf-8,' +
                    encodeURIComponent(JSON.stringify($currentEvent))}
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

    .pre {
        display: flex;
        white-space: pre;
        justify-content: flex-start;
        gap: 20px;
        font-family: monospace;
    }

    .addressStr {
        width: 70px;
    }
</style>
