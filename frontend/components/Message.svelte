<script lang="ts">
    import { untrack } from 'svelte';
    import type { Attachment } from 'svelte/attachments';
    import type { Event } from '../event';
    import { END_REASONS, displayRule, formatPort, getEventById } from '../event';
    import { formatFrameCount } from '../decoded';
    import { currentEvent, filterBySensor, userSensors } from '../store';

    interface Props {
        message: Event;
        follow?: boolean;
        selectable?: boolean;
        selectedIds?: string[];
    }

    let {
        message,
        follow = false,
        selectable = false,
        selectedIds = $bindable([]),
    }: Props = $props();

    let source = $derived(`${message.srcHost}:${message.srcPort}`);
    let destPort = $derived(formatPort(message));
    let handlerLabel = $derived(message.handler || displayRule(message) || '');
    let selected = $derived(
        message.id ? $currentEvent?.id === message.id : $currentEvent === message,
    );
    let matchedSensor = $derived(
        $userSensors.find(
            (s) =>
                s.id === message.sensorID ||
                s.id.startsWith(message.sensorID) ||
                message.sensorID.startsWith(s.id),
        ),
    );
    let sensorName = $derived(matchedSensor?.name);
    let sensorTitle = $derived(
        sensorName
            ? `${sensorName} (${message.sensorID}) - Click to filter`
            : `Filter by sensor: ${message.sensorID}`,
    );

    async function click() {
        if (selectable && message.id) {
            try {
                const full = await getEventById(message.id);
                currentEvent.set(full);
            } catch {
                currentEvent.set(message);
            }
            return;
        }
        currentEvent.set(message);
    }

    function onKeydown(e: KeyboardEvent) {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            click();
        }
    }

    function stopRowActivation(e: MouseEvent) {
        e.stopPropagation();
    }

    function onSensorClick(e: MouseEvent) {
        e.stopPropagation();
        if (message.sensorID) {
            filterBySensor(message.sensorID);
        }
    }

    function onSensorKeydown(e: KeyboardEvent) {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            e.stopPropagation();
            if (message.sensorID) {
                filterBySensor(message.sensorID);
            }
        }
    }

    // untrack so Resume (follow toggled later) does not re-scroll existing rows.
    const scrollIntoViewIfFollowing: Attachment = (element) => {
        if (untrack(() => follow)) {
            element.scrollIntoView();
        }
    };
</script>

{#snippet cells()}
    <span
        class="cell sensor"
        role="button"
        tabindex="0"
        title={sensorTitle}
        onclick={onSensorClick}
        onkeydown={onSensorKeydown}
    >
        {message.sensorID}
    </span>
    <span class="cell source" title={message.srcPtr || source}>{source}</span>
    <span class="cell dest">{destPort}</span>
    <span class="cell handler" title={handlerLabel || undefined}>{handlerLabel}</span>
    <span class="cell scanner" title={message.scanner}>{message.scanner ?? ''}</span>
    <span
        class="cell end"
        title={message.endReason
            ? (END_REASONS[message.endReason] ?? message.endReason)
            : undefined}>{message.endReason ?? ''}</span
    >
    <span class="cell frames" title="received/sent"
        >{formatFrameCount(message.decoded, message.frameCount)}</span
    >
    <span class="cell details">Details</span>
{/snippet}

{#if selectable}
    <div
        class={['message', { selected }]}
        role="button"
        tabindex="0"
        onclick={click}
        onkeydown={onKeydown}
        {@attach scrollIntoViewIfFollowing}
        aria-pressed={selected}
    >
        <span class="cell check">
            {#if message.id}
                <input
                    type="checkbox"
                    value={message.id}
                    bind:group={selectedIds}
                    aria-label="Select event"
                    onclick={stopRowActivation}
                />
            {/if}
        </span>
        {@render cells()}
    </div>
{:else}
    <div
        class={['message', { selected }]}
        role="button"
        tabindex="0"
        onclick={click}
        onkeydown={onKeydown}
        {@attach scrollIntoViewIfFollowing}
        aria-pressed={selected}
    >
        {@render cells()}
    </div>
{/if}

<style>
    .message {
        display: grid;
        grid-template-columns: var(--event-cols);
        column-gap: 6px;
        align-items: center;
        width: 100%;
        margin: 0;
        padding: 1px 0;
        font-family: monospace;
        font-size: 12px;
        line-height: 1.2;
        background: none;
        border: none;
        border-bottom: 1px solid #eee;
        text-align: left;
        cursor: pointer;
    }

    .message:hover,
    .selected {
        background: #f5f5f5;
    }

    .cell {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .cell.sensor {
        cursor: pointer;
    }

    .cell.sensor:hover {
        text-decoration: underline;
        color: #0066cc;
    }

    .check {
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: visible;
    }

    .check input {
        margin: 0;
        cursor: pointer;
    }

    .dest {
        overflow: visible;
        text-overflow: clip;
    }

    .frames {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }

    .details {
        text-decoration: underline;
    }
</style>
