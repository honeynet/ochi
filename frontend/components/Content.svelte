<script lang="ts">
    import { hexy } from 'hexy';
    import { currentEvent, token, isAuthenticated } from '../store';
    import { API_ENDPOINT } from '../constants';
    import type { Event } from '../event';
    import { END_REASONS, displayRule, formatDest, formatDuration, hasTime } from '../event';
    import {
        formatFrameCount,
        framePayload,
        frameTableColumns,
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
    let showClientHello = $state(false);

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
        showClientHello = false;
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
            body: JSON.stringify({ ...eventToShare, id: undefined, ownerID: undefined }),
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
        {#if hasTime($currentEvent.startedAt)}
            Started: {$currentEvent.startedAt}<br />
        {/if}
        Timestamp: {$currentEvent.timestamp}<br />
        {#if $currentEvent.durationMs !== undefined}
            Duration: <span title={`${$currentEvent.durationMs}ms`}
                >{formatDuration($currentEvent.durationMs)}</span
            ><br />
        {/if}
        {#if $currentEvent.endReason}
            End: <span title={END_REASONS[$currentEvent.endReason]}>{$currentEvent.endReason}</span
            ><br />
        {:else}
            End: not set<br />
        {/if}
        {#if $currentEvent.sensorVersion}
            Sensor: {$currentEvent.sensorVersion}<br />
        {:else}
            Sensor: unversioned<br />
        {/if}
        {#if $currentEvent.payloadHash}
            Payload hash: {$currentEvent.payloadHash} (first frame)<br />
        {/if}
        {#if formatFrameCount($currentEvent.decoded, $currentEvent.frameCount)}
            Frames: <span title="received/sent"
                >{formatFrameCount($currentEvent.decoded, $currentEvent.frameCount)}</span
            ><br />
        {/if}
        {#if $currentEvent.srcPtr}
            PTR: {$currentEvent.srcPtr}<br />
        {/if}
        {#if $currentEvent.scanner}
            Scanner: {$currentEvent.scanner}<br />
        {:else if $currentEvent.srcPtr}
            Scanner: unknown (PTR {$currentEvent.srcPtr})<br />
        {/if}
        {#if $currentEvent.tls}
            {@const tls = $currentEvent.tls}
            <div class="tls">
                TLS terminated by sensor (payload and frames are decrypted)<br />
                {#if tls.serverName}
                    SNI: {tls.serverName}<br />
                {/if}
                {#if tls.alpn?.length}
                    ALPN: {tls.alpn.join(', ')}<br />
                {/if}
                {#if tls.version}
                    Version: {tls.version}<br />
                {/if}
                Cipher: {tls.cipher || '(handshake failed)'}<br />
                {#if tls.clientHello}
                    <button type="button" onclick={() => (showClientHello = !showClientHello)}
                        >{showClientHello ? '▾' : '▸'}</button
                    >
                    ClientHello{tls.truncated ? ' (truncated at 4 KiB)' : ''}
                    {#if showClientHello}
                        {@render hexdump(tls.clientHello)}
                    {/if}
                {/if}
            </div>
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
        {#if decodedFrames && decodedFrames.length === 0}
            <p>No frames.</p>
        {:else if decodedFrames}
            <div class="frames">
                <table>
                    <thead>
                        <tr>
                            <th></th>
                            {#each decodedColumns as column (column)}
                                <th>{column}</th>
                            {/each}
                        </tr>
                    </thead>
                    <tbody>
                        {#each decodedFrames as frame, i (i)}
                            <tr class={frame.direction === 'write' ? 'write' : undefined}>
                                <td>
                                    {#if framePayload(frame)}
                                        <button type="button" onclick={() => toggleFrame(i)}
                                            >{expandedFrames.has(i) ? '▾' : '▸'}</button
                                        >
                                    {/if}
                                </td>
                                {#each decodedColumns as column (column)}
                                    <td>{formatFrameValue(frame[column], column)}</td>
                                {/each}
                            </tr>
                            {#if expandedFrames.has(i) && framePayload(frame)}
                                <tr>
                                    <td colspan={decodedColumns.length + 1}>
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
    tr.write {
        background-color: #eef4fb;
    }
    .tls {
        margin: 8px 0;
        padding: 6px 8px;
        border-left: 3px solid #4a7fb5;
        background-color: #f4f8fc;
    }
    th,
    td {
        border: 1px solid #ccc;
        padding: 4px 8px;
        text-align: left;
        vertical-align: top;
    }
</style>
