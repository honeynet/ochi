<script lang="ts">
    import Message from './Message.svelte';
    import EventHeader from './EventHeader.svelte';
    import type { Event } from '../event';
    import { filterEvent } from '../eventFilter';
    import {
        maxNumberOfMessages,
        parsedFilter,
        env,
        recordObservedSensorId,
        eventColumnWidths,
        buildEventColsStyle,
    } from '../store';
    import { ENV_PROD } from '../constants';
    import { untrack } from 'svelte';

    let messages = $state<Event[]>([]);
    let follow = $state(true);

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

            if ($maxNumberOfMessages < messages.length) {
                messages = messages.slice(messages.length - $maxNumberOfMessages);
            }
        }
    }
</script>

<div class="column">
    <div
        id="message-log"
        style="--event-cols: {buildEventColsStyle($eventColumnWidths)};"
        onwheel={() => {
            follow = false;
        }}
    >
        <EventHeader />
        {#each messages as message (message.timestamp)}
            <Message {message} {follow} />
        {/each}
    </div>

    {#if !follow}
        <button
            onclick={() => {
                follow = true;
            }}
            id="resume-btn">Resume</button
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
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        overflow-x: auto;
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
