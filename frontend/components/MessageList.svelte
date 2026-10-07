<script lang="ts">
    import Message from './Message.svelte';
    import type { Event } from '../event';
    import { filterEvent } from '../eventFilter';
    import { maxNumberOfMessages, parsedFilter, env } from '../store';
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
        onwheel={() => {
            follow = false;
        }}
    >
        <div class="event-head">
            <span>Sensor</span>
            <span>Source</span>
            <span>Port</span>
            <span>Handler</span>
            <span>Scanner</span>
            <span>End</span>
            <span class="frames" title="received/sent">Frames</span>
            <span></span>
        </div>
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
        --event-cols: 8ch minmax(14ch, 1.4fr) 9ch 8ch minmax(8ch, 0.9fr) 20ch 6ch 7ch;
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        overflow-x: hidden;
        scrollbar-gutter: stable;
    }

    .event-head {
        display: grid;
        grid-template-columns: var(--event-cols);
        column-gap: 6px;
        position: sticky;
        top: 0;
        z-index: 1;
        padding: 0 0 2px;
        font-family: monospace;
        font-size: 11px;
        font-weight: 600;
        line-height: 1.2;
        color: #555;
        background: #fff;
        border-bottom: 1px solid #ccc;
    }

    .event-head span {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .frames {
        text-align: right;
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
