<svelte:options runes={true} />

<script lang="ts">
    import { onMount } from 'svelte';
    import { url } from '@roxi/routify';
    import { token } from '../store';
    import { type Event, deleteEvent, displayRule, formatPort, getEvents } from '../event';

    let events = $state<Event[]>([]);
    let error = $state('');
    let loading = $state(true);

    onMount(() => {
        reloadEvents();
    });

    async function reloadEvents() {
        try {
            events = await getEvents($token);
            error = '';
        } catch (e) {
            error = e instanceof Error ? e.message : 'Could not fetch events';
        } finally {
            loading = false;
        }
    }

    async function deleteAndReload(id?: string) {
        if (!id) {
            console.warn('Cannot delete event without an id');
            return;
        }
        try {
            await deleteEvent(id, $token);
        } catch (e) {
            error = e instanceof Error ? e.message : 'Could not delete an event';
            return;
        }
        await reloadEvents();
    }
</script>

<h2 class="eventList__title">Saved Events</h2>
{#if loading}
    <p class="eventList__empty">Loading...</p>
{:else if error}
    <p class="eventList__empty">{error}</p>
{:else if events.length === 0}
    <p class="eventList__empty">No saved events</p>
{/if}
<ul class="eventList__items">
    {#each events as event, index (event.id)}
        <li class="eventList__item">
            <div class="eventList__item-container">
                {index + 1}.
                <div class="eventList__info-container">
                    <p class="eventList__info-field eventList__content">
                        {event.srcHost}:{event.srcPort} to {formatPort(event)}
                    </p>
                    <p class="eventList__info-field eventList__meta">
                        {event.timestamp}
                        {#if displayRule(event)}| {displayRule(event)}{/if}
                    </p>
                </div>
            </div>
            <div class="eventList__buttons-container">
                <a href={$url('/events/:id', { id: event.id })}>Open</a>
                <button onclick={() => deleteAndReload(event.id)}>Delete</button>
            </div>
        </li>
    {/each}
</ul>

<style>
    .eventList__items {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 15px;
    }

    .eventList__item {
        width: 40%;
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
    }

    .eventList__item-container {
        display: flex;
        justify-content: flex-start;
        gap: 10px;
        font-size: 20px;
    }

    .eventList__info-field {
        margin: 0;
    }

    .eventList__meta {
        font-style: italic;
        font-size: 14px;
    }

    .eventList__info-container {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .eventList__buttons-container {
        align-self: center;
        display: flex;
        gap: 10px;
        align-items: center;
    }

    .eventList__title,
    .eventList__empty {
        text-align: center;
    }
</style>
