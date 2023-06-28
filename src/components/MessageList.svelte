<script lang="ts">
    import Message from './Message.svelte';
    import type { Event } from '../event';
    import { filterEvent } from '../eventFilter';
    import { maxNumberOfMessages, parsedFilter } from '../store';

    let messages: Event[] = [];
    let follow: boolean = true;

    parsedFilter.subscribe((value) => {
        if (value) {
            messages = messages.filter((message) => filterEvent(message, value));
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

<div
    class="column"
    id="message-log"
    on:wheel={() => {
        follow = false;
    }}
>
    {#each messages as message (message.timestamp)}
        <Message on:message {message} {follow} />
    {/each}
</div>

{#if !follow}
    <button
        on:click={() => {
            follow = true;
        }}
        id="resume-btn">Resume</button
    >
{/if}

<style>
    .column {
        flex: 50%;
        padding: 15px 20px;
    }

    #message-log {
        width: 100%;
        flex-grow: 1;
        overflow-y: scroll;
    }

    #resume-btn {
        position: fixed;
        bottom: 2rem;
        z-index: 2;
        left: 40vw;
        cursor: pointer;
    }
</style>
