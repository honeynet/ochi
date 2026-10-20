<svelte:options runes={true} />

<script lang="ts">
    import { onMount } from 'svelte';
    import { get } from 'svelte/store';
    import { token, currentEvent, eventColumnWidths, buildEventColsStyle } from '../store';
    import { type Event, deleteEvents, getEventById, getEvents } from '../event';
    import Message from './Message.svelte';
    import EventHeader from './EventHeader.svelte';
    import { showError } from '../toast';

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

    async function selectEvent(event: Event) {
        if (event.id) {
            try {
                const full = await getEventById(event.id);
                currentEvent.set(full);
                return;
            } catch {
                // fall through
            }
        }
        currentEvent.set(event);
    }

    function onWindowKeydown(e: KeyboardEvent) {
        if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') {
            return;
        }
        if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) {
            return;
        }
        if (isTypingTarget(e.target)) {
            return;
        }
        if (events.length === 0) {
            return;
        }

        e.preventDefault();

        const cur = get(currentEvent);
        let idx = cur?.id ? events.findIndex((ev) => ev.id === cur.id) : -1;
        if (idx === -1) {
            selectEvent(events[events.length - 1]);
            return;
        }

        const next = e.key === 'ArrowUp' ? idx - 1 : idx + 1;
        if (next < 0 || next >= events.length) {
            return;
        }
        selectEvent(events[next]);
    }

    function syncCurrentEvent() {
        const cur = get(currentEvent);
        if (!cur?.id) {
            return;
        }
        if (!events.some((e) => e.id === cur.id)) {
            currentEvent.set(undefined);
        }
    }

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
            syncCurrentEvent();
        } catch (e) {
            if (id === requestId) {
                error = e instanceof Error ? e.message : 'Could not fetch events';
                showError(e, 'Could not fetch events');
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

    function toggleAll() {
        selected = allSelected
            ? selected.filter((id) => !pageIds.includes(id))
            : [...new Set([...selected, ...pageIds])];
    }

    async function deleteSelected() {
        const toDelete = [...selected];
        try {
            await deleteEvents(toDelete, $token);
        } catch (e) {
            error = e instanceof Error ? e.message : 'Could not delete events';
            showError(e, 'Could not delete events');
            return;
        }
        selected = [];
        const cur = get(currentEvent);
        if (cur?.id && toDelete.includes(cur.id)) {
            currentEvent.set(undefined);
        }
        await reloadEvents();
    }
</script>

<svelte:window onkeydown={onWindowKeydown} />

<div class="column">
    <div class="toolbar">
        <span class="toolbar-label">Saved Events</span>
        {#if total > 0}
            <label class="select-page">
                <input type="checkbox" checked={allSelected} onchange={toggleAll} />
                Select page
            </label>
            <button disabled={selected.length === 0} onclick={deleteSelected}>
                Delete selected ({selected.length})
            </button>
        {/if}
    </div>

    {#if loading}
        <p class="status">Loading...</p>
    {:else if error}
        <p class="status">{error}</p>
    {:else if total === 0}
        <p class="status">No saved events</p>
    {:else}
        <div
            id="message-log"
            style="--event-cols: {buildEventColsStyle($eventColumnWidths, true)};"
        >
            <EventHeader selectable />
            {#each events as event (event.id)}
                <Message message={event} follow={false} selectable bind:selectedIds={selected} />
            {/each}
        </div>
    {/if}

    {#if pageCount > 1}
        <div class="pager">
            <button disabled={page === 0} onclick={() => goTo(page - 1)}>Previous</button>
            <span>Page {page + 1} of {pageCount}</span>
            <button disabled={page >= pageCount - 1} onclick={() => goTo(page + 1)}>Next</button>
        </div>
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

    .toolbar {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 12px;
        margin-bottom: 6px;
        font-size: 12px;
    }

    .toolbar-label {
        font-weight: 600;
        margin-right: auto;
    }

    .select-page {
        display: flex;
        align-items: center;
        gap: 4px;
        cursor: pointer;
    }

    .status {
        margin: 12px 0;
        font-family: monospace;
        font-size: 12px;
        color: #555;
    }

    #message-log {
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        overflow-x: auto;
        scrollbar-gutter: stable;
    }

    .pager {
        display: flex;
        justify-content: center;
        align-items: center;
        gap: 15px;
        padding-top: 8px;
        font-size: 12px;
    }
</style>
