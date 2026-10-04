<script lang="ts">
    import { hexy } from 'hexy';
    import { currentEvent, token, isAuthenticated } from '../store';
    import { API_ENDPOINT } from '../constants';
    import type { Event } from '../event';
    import { displayRule, formatDest } from '../event';
    import {
        framePayload,
        frameTableColumns,
        formatFrameHeader,
        formatFrameValue,
        isFrameArray,
    } from '../decoded';
    import { url } from '@roxi/routify';
    import { copy } from 'svelte-copy';
    import { SvelteSet } from 'svelte/reactivity';

    interface Props {
        isShared: boolean;
    }

    let { isShared }: Props = $props();

    type RenderResult = { name: string; content: string[] };

    const HEX_BYTES_PER_LINE = 24;
    // hexy default grouping is 2 bytes (4 hex chars). Address is 8 chars + ": "
    // (10); hex column is 2.5 chars per byte, then 1 space before ASCII.
    const HEX_COLUMN_END = 10 + Math.floor(HEX_BYTES_PER_LINE * 2.5) + 1;

    function render(payload: string): RenderResult[] {
        const result = hexy(atob(payload), { width: HEX_BYTES_PER_LINE });
        const resultLines = result.split('\n');
        let addressStr = '';
        let hexStr = '';
        let plainStr = '';
        resultLines.forEach((item, idx) => {
            if (item) {
                addressStr += (idx > 0 ? '\n' : '') + item.substring(0, item.indexOf(':'));
                hexStr +=
                    (idx > 0 ? '\n' : '') + item.substring(item.indexOf(':') + 2, HEX_COLUMN_END);
                plainStr += (idx > 0 ? '\n' : '') + item.substring(HEX_COLUMN_END);
            }
        });
        return [
            { name: 'addressStr', content: addressStr.split('\n') },
            { name: 'hexStr', content: hexStr.split('\n') },
            { name: 'plainStr', content: plainStr.split('\n') },
        ];
    }

    let eventCreated = $state<Event | undefined>(undefined);
    let disabled = $state(false);
    let expandedFrames = new SvelteSet<number>();

    let decodedFrames = $derived(
        $currentEvent && isFrameArray($currentEvent.decoded) ? $currentEvent.decoded : null,
    );
    let decodedColumns = $derived(
        decodedFrames ? frameTableColumns(decodedFrames, $currentEvent?.handler) : [],
    );
    let ruleLabel = $derived($currentEvent ? displayRule($currentEvent) : undefined);
    let eventIdentity = $derived(
        $currentEvent
            ? `${$currentEvent.timestamp}|${$currentEvent.srcHost}|${$currentEvent.srcPort}|${$currentEvent.dstPort}|${$currentEvent.payload ?? ''}`
            : '',
    );

    let lastIdentity = '';
    $effect(() => {
        const id = eventIdentity;
        if (id === lastIdentity) {
            return;
        }
        lastIdentity = id;
        eventCreated = undefined;
        disabled = false;
        expandedFrames.clear();
    });

    function toggleFrame(index: number) {
        if (expandedFrames.has(index)) {
            expandedFrames.delete(index);
        } else {
            expandedFrames.add(index);
        }
    }

    async function createEvent() {
        if (!$currentEvent) {
            throw new Error('No event selected to share');
        }
        console.log('saving event');
        const eventToShare = $currentEvent;
        const res = await fetch(`${API_ENDPOINT}/api/events`, {
            method: 'POST',
            headers: {
                Authorization: `Bearer ${$token}`,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                payload: eventToShare.payload,
                dstPort: eventToShare.dstPort,
                rule: eventToShare.rule,
                handler: eventToShare.handler,
                transport: eventToShare.transport,
                scanner: eventToShare.scanner,
                sensorID: eventToShare.sensorID,
                srcHost: eventToShare.srcHost,
                srcPort: eventToShare.srcPort,
                timestamp: eventToShare.timestamp,
                decoded: eventToShare.decoded,
                startedAt: eventToShare.startedAt,
                durationMs: eventToShare.durationMs,
                srcPtr: eventToShare.srcPtr,
                dstHost: eventToShare.dstHost,
                sensorVersion: eventToShare.sensorVersion,
                ruleName: eventToShare.ruleName,
                payloadHash: eventToShare.payloadHash,
                frameCount: eventToShare.frameCount,
                endReason: eventToShare.endReason,
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

    async function share() {
        disabled = true;
        try {
            eventCreated = await createEvent();
        } catch (error) {
            disabled = false;
            throw error;
        }
    }

    function downloadEvent() {
        if (!$currentEvent) {
            return;
        }
        const jsonData = JSON.stringify($currentEvent);
        const blob = new Blob([jsonData], { type: 'application/json' });
        const objectUrl = window.URL.createObjectURL(blob);

        const a = document.createElement('a');
        a.style.display = 'none';
        a.href = objectUrl;
        a.download = 'event.json';

        document.body.appendChild(a);
        a.click();

        window.URL.revokeObjectURL(objectUrl);
        document.body.removeChild(a);
    }
</script>

{#snippet hexdump(payload: string)}
    {@const results = render(payload)}
    <div class="pre">
        {#each results as renderResult (renderResult.name)}
            <div class={renderResult.name}>
                {#each renderResult.content as content, i (`${renderResult.name}-${i}`)}
                    <div class={i % 2 == 0 ? 'even' : 'odd'}>{content}</div>
                {/each}
            </div>
        {/each}
    </div>
{/snippet}

<div class="column" id="content">
    {#if $currentEvent}
        <span title={$currentEvent.srcPtr || ''}>{$currentEvent.srcHost}</span
        >:{$currentEvent.srcPort}
        -> {formatDest($currentEvent)}<br />
        {#if $currentEvent.handler}
            Handler: {$currentEvent.handler}<br />
        {/if}
        {#if ruleLabel}
            {ruleLabel}<br />
        {/if}
        {#if $currentEvent.startedAt}
            Started: {$currentEvent.startedAt}<br />
        {/if}
        Timestamp: {$currentEvent.timestamp}<br />
        {#if $currentEvent.durationMs !== undefined}
            Duration: {$currentEvent.durationMs}ms<br />
        {/if}
        {#if $currentEvent.endReason}
            End: {$currentEvent.endReason}<br />
        {:else}
            End: (old handler)<br />
        {/if}
        {#if $currentEvent.sensorVersion}
            Sensor: {$currentEvent.sensorVersion}<br />
        {:else}
            Sensor: unversioned<br />
        {/if}
        {#if $currentEvent.payloadHash}
            Payload hash: {$currentEvent.payloadHash} (first frame)<br />
        {/if}
        {#if $currentEvent.frameCount !== undefined}
            Frames: {$currentEvent.frameCount}<br />
        {/if}
        {#if $currentEvent.srcPtr}
            PTR: {$currentEvent.srcPtr}<br />
        {/if}
        {#if $currentEvent.scanner}
            Scanner: {$currentEvent.scanner}<br />
        {:else if $currentEvent.srcPtr}
            Scanner: unknown (PTR {$currentEvent.srcPtr})<br />
        {/if}
        {#if $currentEvent.payload}
            Payload:
            {@render hexdump($currentEvent.payload)}
        {/if}
        {#if !isShared}
            <button onclick={downloadEvent}>Download</button>
            {#if !eventCreated}
                <button disabled={!$isAuthenticated || disabled} onclick={share}>Share</button>
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
        {#if decodedFrames}
            <div class="frames">
                <table>
                    <thead>
                        <tr>
                            <th></th>
                            {#each decodedColumns as column (column)}
                                <th>{column}</th>
                            {/each}
                            {#if $currentEvent.handler === 'smb'}
                                <th>header</th>
                            {/if}
                        </tr>
                    </thead>
                    <tbody>
                        {#each decodedFrames as frame, i (i)}
                            <tr>
                                <td>
                                    {#if framePayload(frame)}
                                        <button type="button" onclick={() => toggleFrame(i)}
                                            >{expandedFrames.has(i) ? '▾' : '▸'}</button
                                        >
                                    {/if}
                                </td>
                                {#each decodedColumns as column (column)}
                                    <td>{formatFrameValue(frame[column])}</td>
                                {/each}
                                {#if $currentEvent.handler === 'smb'}
                                    <td>{formatFrameHeader(frame.header)}</td>
                                {/if}
                            </tr>
                            {#if expandedFrames.has(i) && framePayload(frame)}
                                <tr>
                                    <td colspan="20">
                                        {@render hexdump(framePayload(frame)!)}
                                    </td>
                                </tr>
                            {/if}
                        {/each}
                    </tbody>
                </table>
            </div>
        {:else if $currentEvent.decoded}
            <div class="payload">
                {JSON.stringify($currentEvent.decoded, null, 2)}
            </div>
        {/if}
    {/if}
</div>

<style>
    .column {
        min-width: 0;
        padding: 15px 20px;
        box-sizing: border-box;
    }

    .pre {
        display: flex;
        justify-content: flex-start;
        gap: 20px;
        overflow-x: auto;
        font-family: monospace;
        white-space: pre;
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
    .frames {
        padding-top: 20px;
        overflow-x: auto;
    }
    table {
        border-collapse: collapse;
        font-family: monospace;
        font-size: 12px;
    }
    th,
    td {
        border: 1px solid #ccc;
        padding: 4px 8px;
        text-align: left;
        vertical-align: top;
    }
</style>
