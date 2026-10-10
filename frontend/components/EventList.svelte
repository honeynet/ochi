<svelte:options runes={true} />

<script lang="ts">
    import { onMount } from 'svelte';
    import { url } from '@roxi/routify';
    import { token } from '../store';
    import {
        type Event,
        deleteEvent,
        deleteEvents,
        displayRule,
        formatPort,
        getEvents,
    } from '../event';

    const PAGE_SIZE = 20;

    let events = $state<Event[]>([]);
    let total = $state(0);
    let error = $state('');
    let loading = $state(true);
    let page = $state(0);
    let selected = $state<string[]>([]);

    let pageCount = $derived(Math.max(1, Math.ceil(total / PAGE_SIZE)));
    let pageIds = $derived(events.flatMap((e) => (e.id ? [e.id] : [])));
    let allSelected = $derived(pageIds.length > 0 && pageIds.every((id) => selected.includes(id)));

    onMount(() => {
        reloadEvents();
    });

    let requestId = 0;

    async function reloadEvents(target = page) {
        const id = ++requestId;
        try {
            let result = await getEvents($token, PAGE_SIZE, target * PAGE_SIZE);
            const lastPage = Math.max(0, Math.ceil(result.total / PAGE_SIZE) - 1);
            if (target > lastPage) {
                target = lastPage;
                result = await getEvents($token, PAGE_SIZE, target * PAGE_SIZE);
            }
            if (id !== requestId) {
                return;
            }
            events = result.events;
            total = result.total;
            page = target;
            error = '';
        } catch (e) {
            if (id === requestId) {
                error = e instanceof Error ? e.message : 'Could not fetch events';
            }
        } finally {
            if (id === requestId) {
                loading = false;
            }
        }
    }

    function goTo(next: number) {
        selected = [];
        reloadEvents(next);
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
        selected = selected.filter((s) => s !== id);
        await reloadEvents();
    }

    function toggleAll() {
        selected = allSelected
            ? selected.filter((id) => !pageIds.includes(id))
            : [...new Set([...selected, ...pageIds])];
    }

    async function deleteSelected() {
        try {
            await deleteEvents(selected, $token);
        } catch (e) {
            error = e instanceof Error ? e.message : 'Could not delete events';
            return;
        }
        selected = [];
        await reloadEvents();
    }
</script>

<h2 class="eventList__title">Saved Events</h2>
{#if loading}
    <p class="eventList__empty">Loading...</p>
{:else if error}
    <p class="eventList__empty">{error}</p>
{:else if total === 0}
    <p class="eventList__empty">No saved events</p>
{/if}
{#if total > 0}
    <div class="eventList__toolbar">
        <label>
            <input type="checkbox" checked={allSelected} onchange={toggleAll} />
            Select page
        </label>
        <button disabled={selected.length === 0} onclick={deleteSelected}>
            Delete selected ({selected.length})
        </button>
    </div>
{/if}
<ul class="eventList__items">
    {#each events as event, index (event.id)}
        <li class="eventList__item">
            <div class="eventList__item-container">
                {#if event.id}
                    <input type="checkbox" value={event.id} bind:group={selected} />
                {/if}
                {page * PAGE_SIZE + index + 1}.
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
{#if pageCount > 1}
    <div class="eventList__pager">
        <button disabled={page === 0} onclick={() => goTo(page - 1)}>Previous</button>
        <span>Page {page + 1} of {pageCount}</span>
        <button disabled={page >= pageCount - 1} onclick={() => goTo(page + 1)}>Next</button>
    </div>
{/if}

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

    .eventList__toolbar,
    .eventList__pager {
        display: flex;
        justify-content: center;
        align-items: center;
        gap: 15px;
    }

    .eventList__title,
    .eventList__empty {
        text-align: center;
    }
</style>
