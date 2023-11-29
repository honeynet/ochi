<script lang="ts">
    import { hexy } from 'hexy';
    import { currentEvent, token, isAuthenticated } from '../store';
    import { API_ENDPOINT } from '../constants';
    import type { Event } from '../event';
    import { url } from '@roxi/routify';
    import { copy } from 'svelte-copy';

    export let isShared: boolean;

    function render(payload: string) {
        const result = hexy(atob(payload), { width: 8 });
        const resultLines = result.split('\n');
        let addressStr = '';
        let hexStr = '';
        let plainStr = '';
        resultLines.forEach((item, idx) => {
            if (item) {
                addressStr += (idx > 0 ? '\n' : '') + item.substring(0, item.indexOf(':'));
                hexStr += (idx > 0 ? '\n' : '') + item.substring(item.indexOf(':') + 2, 31);
                plainStr += (idx > 0 ? '\n' : '') + item.substring(31);
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
        eventCreated = undefined;
    }

    async function createEvent() {
        console.log('saving event');
        const res = await fetch(`${API_ENDPOINT}/api/events`, {
            method: 'POST',
            headers: {
                Authorization: `Bearer ${$token}`,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                payload: $currentEvent.payload,
                dstPort: $currentEvent.dstPort,
                rule: $currentEvent.rule,
                handler: $currentEvent.handler,
                transport: $currentEvent.transport,
                scanner: $currentEvent.scanner,
                sensorID: $currentEvent.sensorID,
                srcHost: $currentEvent.srcHost,
                srcPort: $currentEvent.srcPort,
                timestamp: $currentEvent.timestamp,
                decoded: $currentEvent.decoded,
            }),
        });

        if (res.ok) {
            console.log('received success');
            const event = await res.json();
            return event;
        } else {
            console.log('failed to save ' + res.text());
            throw new Error('Could not create an event');
        }
    }

    async function getEventById(id): Promise<Event> {
        console.log('fetching event');
        const res = await fetch(`${API_ENDPOINT}/api/events/${id}`, {
            method: 'GET',
            headers: {
                Authorization: `Bearer ${$token}`,
                'Content-Type': 'application/json',
            },
        });

        if (res.ok) {
            console.log('received success ');
            const event = await res.json();
            return event;
        } else {
            console.log('failed to save ' + res.text());
            throw new Error('Could not fetch an event');
        }
    }

    async function getEvents(): Promise<Event[]> {
        console.log('fetching queries');
        const res = await fetch(`${API_ENDPOINT}/api/events`, {
            method: 'GET',
            headers: {
                Authorization: `Bearer ${$token}`,
                'Content-Type': 'application/json',
            },
        });

        if (res.ok) {
            console.log('received success ');
            const data = await res.json();
            return data;
        } else {
            console.log('failed to get ' + res.text());
            throw new Error('Could not fetch events');
        }
    }

    let eventCreated: Event;
    let disabled = false;
    async function share() {
        await createEvent().then((event) => {
            eventCreated = event;
            disabled = true;
        });
    }

    function downloadEvent() {
        const jsonData = JSON.stringify($currentEvent);
        const blob = new Blob([jsonData], { type: 'application/json' });
        const url = window.URL.createObjectURL(blob);

        const a = document.createElement('a');
        a.style.display = 'none';
        a.href = url;
        a.download = 'event.json';

        document.body.appendChild(a);
        a.click();

        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
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
        {/if}
        {#if !isShared}
            <button on:click={downloadEvent}>Download</button>
            {#if !eventCreated}
                <button disabled={!$isAuthenticated} on:click={share}>Share</button>
            {:else}
                <p>
                    Event is created which you can view <a
                        target="blank"
                        href={$url('/events/:id', { id: eventCreated.id })}>here</a
                    >
                    or
                    <button
                        use:copy={window.location.protocol +
                            '//' +
                            window.location.host +
                            $url('/events/:id', { id: eventCreated.id })}>copy url</button
                    >.
                </p>
            {/if}
        {/if}
        {#if $currentEvent.decoded}
            <div class="payload">
                {JSON.stringify($currentEvent.decoded, null, 2)}
            </div>
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
        min-width: 355px;
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
    .payload {
        display: block;
        unicode-bidi: embed;
        font-family: monospace;
        white-space: pre;
        text-wrap: wrap;
        padding-top: 20px;
        word-break: break-all;
    }
</style>
