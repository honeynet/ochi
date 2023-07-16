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
                addressStr += (idx > 0 ? '\n' : '') + item.substring(0, item.indexOf(':'));
                hexStr += (idx > 0 ? '\n' : '') + item.substring(item.indexOf(':') + 2, 51);
                plainStr += (idx > 0 ? '\n' : '') + item.substring(51);
            }
        });
        return [
            { name: 'addressStr', content: addressStr.split('\n') },
            { name: 'hexStr', content: hexStr.split('\n') },
            { name: 'plainStr', content: plainStr.split('\n') },
        ];
    }

    let renderResults;
    $: if ($currentEvent && $currentEvent.payload) {
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
            Payload:
            <div class="pre">
                {#each renderResults as renderResult}
                    <div class={renderResult.name}>
                        {#each renderResult.content as content, i}
                            <div class={i % 2 == 0 ? 'even' : 'odd'}>{content}</div>
                        {/each}
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
        justify-content: flex-start;
        gap: 20px;
        min-width: 535px;
        font-family: monospace;
        padding-top: 15px;
        padding-bottom: 15px;
    }

    .even {
        background-color: #fafafa;
    }

    .odd {
        background-color: #d3d3d3;
    }
</style>
