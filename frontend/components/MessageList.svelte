<script lang="ts">
    import Message from './Message.svelte';
    import EventHeader from './EventHeader.svelte';
    import type { Event } from '../event';
    import { filterEvent } from '../eventFilter';
    import {
        currentEvent,
        maxNumberOfMessages,
        parsedFilter,
        env,
        recordObservedSensorId,
        eventColumnWidths,
        hasCustomColumnWidths,
        buildEventColsStyle,
    } from '../store';
    import { ENV_PROD } from '../constants';
    import { get } from 'svelte/store';
    import { untrack } from 'svelte';

    let { active = true }: { active?: boolean } = $props();

    let messages = $state<Event[]>([]);
    let follow = $state(true);
    // Events that arrived while the list was paused, shown on the Resume button.
    let missed = $state(0);

    function isTypingTarget(target: EventTarget | null): boolean {
        if (!(target instanceof HTMLElement)) {
            return false;
        }
        const tag = target.tagName;
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') {
            return true;
        }
        if (target.isContentEditable) {
            return true;
        }
        return !!target.closest('[role="separator"]');
    }

    function eventIndex(list: Event[], cur: Event | undefined): number {
        if (!cur) {
            return -1;
        }
        return list.findIndex((m) => (m.id ? m.id === cur.id : m === cur));
    }

    function onWindowKeydown(e: KeyboardEvent) {
        if (!active) {
            return;
        }
        if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') {
            return;
        }
        if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) {
            return;
        }
        if (isTypingTarget(e.target)) {
            return;
        }
        if (messages.length === 0) {
            return;
        }

        e.preventDefault();
        follow = false;

        let idx = eventIndex(messages, get(currentEvent));
        if (idx === -1) {
            currentEvent.set(messages[messages.length - 1]);
            return;
        }

        const next = e.key === 'ArrowUp' ? idx - 1 : idx + 1;
        if (next < 0 || next >= messages.length) {
            return;
        }
        currentEvent.set(messages[next]);
    }

    $effect(() => {
        const value = $parsedFilter;
        if (value) {
            untrack(() => {
                messages = messages.filter((message) => filterEvent(message, value));
            });
        }
    });

    $effect(() => {
        const value = $maxNumberOfMessages;
        untrack(() => {
            if (value < messages.length) {
                messages = messages.slice(messages.length - value, messages.length);
            }
        });
    });

    $effect(() => {
        if ($env == ENV_PROD) {
            untrack(() => {
                messages = [];
            });
        }
    });

    export function onNewMessage(message: Event) {
        if (message.sensorID) {
            recordObservedSensorId(message.sensorID);
        }
        if (!$parsedFilter || filterEvent(message, $parsedFilter)) {
            messages.push(message);
            messages = messages;
            if (!follow) {
                missed++;
            }

            if ($maxNumberOfMessages < messages.length) {
                messages = messages.slice(messages.length - $maxNumberOfMessages);
            }
        }
    }
</script>

<svelte:window onkeydown={onWindowKeydown} />

<div class="column">
    <div
        id="message-log"
        style={$hasCustomColumnWidths
            ? `--event-cols: ${buildEventColsStyle($eventColumnWidths)};`
            : undefined}
        onwheel={() => {
            follow = false;
        }}
    >
        <EventHeader />
        {#each messages as message (message)}
            <Message {message} {follow} />
        {/each}
    </div>

    {#if !follow}
        <button
            onclick={() => {
                follow = true;
                missed = 0;
            }}
            id="resume-btn">Resume{missed > 0 ? ` (${missed} new)` : ''}</button
        >
    {/if}
</div>

<style>
    .column {
        position: relative;
        display: flex;
        flex-direction: column;
        flex: 1;
        min-width: 0;
        min-height: 0;
        padding: 8px 12px;
    }

    #message-log {
        --event-cols: 8ch 8ch minmax(14ch, 1.6fr) 9ch 8ch minmax(6ch, 0.9fr) minmax(11ch, 13ch) 6ch
            7ch;
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        overflow-x: hidden;
        scrollbar-gutter: stable;
    }

    #resume-btn {
        position: absolute;
        bottom: 1rem;
        left: 50%;
        z-index: 2;
        transform: translateX(-50%);
        cursor: pointer;
    }
</style>
