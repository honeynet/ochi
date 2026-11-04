<svelte:options runes={true} />

<script lang="ts">
    import { onMount } from 'svelte';
    import { get } from 'svelte/store';
    import { token, currentEvent } from '../store';
    import { type Event, deleteEvents, getEventById, getEvents } from '../event';
    import Message from './Message.svelte';
    import ConfirmDialog from './ConfirmDialog.svelte';
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

    // Deleting saved events cannot be undone, so it goes through a confirmation dialog.
    let confirmDialog = $state<ConfirmDialog | null>(null);
    let deleting = $state(false);

    function askToDeleteSelected() {
        confirmDialog?.showModal();
    }

    function closeConfirm() {
        confirmDialog?.close();
    }

    async function confirmDeleteSelected() {
        const toDelete = [...selected];
        deleting = true;
        try {
            await deleteEvents(toDelete, $token);
            selected = [];
            const cur = get(currentEvent);
            if (cur?.id && toDelete.includes(cur.id)) {
                currentEvent.set(undefined);
            }
            closeConfirm();
            await reloadEvents();
        } catch (e) {
            showError(e, 'Could not delete events');
        } finally {
            deleting = false;
        }
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
            <button disabled={selected.length === 0 || deleting} onclick={askToDeleteSelected}>
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
        <div id="message-log">
            <div class="event-head">
                <span></span>
                <span>Time</span>
                <span>Sensor</span>
                <span>Source</span>
                <span>Port</span>
                <span>Handler</span>
                <span>Scanner</span>
                <span>End</span>
                <span class="frames" title="received/sent">Frames</span>
                <span></span>
            </div>
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

<ConfirmDialog
    bind:this={confirmDialog}
    confirmText={deleting ? 'Deleting...' : 'Delete'}
    busy={deleting}
    onconfirm={confirmDeleteSelected}
>
    <p>
        Delete <strong>{selected.length}</strong> selected {selected.length === 1
            ? 'event'
            : 'events'}? This action cannot be undone.
    </p>
</ConfirmDialog>

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
        /* See MessageList.svelte: capped tracks, with the slack in the empty track at the end. */
        --event-cols: 2ch 8ch 8ch minmax(14ch, 26ch) 9ch 8ch minmax(6ch, 12ch) minmax(11ch, 13ch)
            6ch 7ch minmax(0, 1fr);
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        overflow-x: hidden;
        scrollbar-gutter: stable;
    }

    .event-head {
        display: grid;
        grid-template-columns: var(--event-cols);
        column-gap: 5px;
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

    .pager {
        display: flex;
        justify-content: center;
        align-items: center;
        gap: 15px;
        padding-top: 8px;
        font-size: 12px;
    }
</style>
