<svelte:options runes={true} />

<script lang="ts">
    import { onDestroy } from 'svelte';
    import {
        eventColumnWidths,
        resetColumnWidth,
        clampColumnWidth,
        type EventColumnWidths,
    } from '../store';

    interface Props {
        selectable?: boolean;
    }

    let { selectable = false }: Props = $props();

    let activeResizer = $state<keyof EventColumnWidths | null>(null);
    let startX = 0;
    let startWidth = 0;

    interface ColumnDef {
        key: keyof EventColumnWidths;
        label: string;
        title?: string;
        className?: string;
    }

    const COLUMNS: ColumnDef[] = [
        { key: 'time', label: 'Time' },
        { key: 'sensor', label: 'Sensor' },
        { key: 'source', label: 'Source' },
        { key: 'port', label: 'Port' },
        { key: 'handler', label: 'Handler' },
        { key: 'scanner', label: 'Scanner' },
        { key: 'end', label: 'End' },
        { key: 'frames', label: 'Frames', title: 'received/sent', className: 'frames' },
        { key: 'details', label: '' },
    ];

    function startResize(col: keyof EventColumnWidths, event: PointerEvent) {
        event.preventDefault();
        event.stopPropagation();
        const handle = event.currentTarget;
        if (handle instanceof HTMLElement && typeof handle.setPointerCapture === 'function') {
            handle.setPointerCapture(event.pointerId);
        }
        activeResizer = col;
        startX = event.clientX;
        startWidth = $eventColumnWidths[col];
        if (typeof document !== 'undefined') {
            document.body.style.cursor = 'col-resize';
            document.body.style.userSelect = 'none';
        }
    }

    function onResizeMove(event: PointerEvent) {
        if (!activeResizer) return;
        const delta = event.clientX - startX;
        const newWidth = clampColumnWidth(activeResizer, startWidth + delta);
        eventColumnWidths.update((w) => ({
            ...w,
            [activeResizer!]: newWidth,
        }));
    }

    function onResizeEnd(event: PointerEvent) {
        if (!activeResizer) return;
        const handle = event.currentTarget;
        if (
            handle instanceof HTMLElement &&
            typeof handle.releasePointerCapture === 'function' &&
            handle.hasPointerCapture?.(event.pointerId)
        ) {
            handle.releasePointerCapture(event.pointerId);
        }
        activeResizer = null;
        if (typeof document !== 'undefined') {
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        }
    }

    onDestroy(() => {
        if (typeof document !== 'undefined') {
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        }
    });
</script>

<div class="event-head">
    {#if selectable}
        <span class="head-cell check"></span>
    {/if}
    {#each COLUMNS as col, i (col.key)}
        <span class={['head-cell', col.className]}>
            <span class="col-title" title={col.title}>{col.label}</span>
            {#if i < COLUMNS.length - 1}
                <span
                    class={['col-resizer', { active: activeResizer === col.key }]}
                    role="separator"
                    aria-orientation="vertical"
                    title="Drag to resize, double-click to reset"
                    onpointerdown={(e) => startResize(col.key, e)}
                    onpointermove={onResizeMove}
                    onpointerup={onResizeEnd}
                    onpointercancel={onResizeEnd}
                    onlostpointercapture={onResizeEnd}
                    ondblclick={(e) => {
                        e.stopPropagation();
                        resetColumnWidth(col.key);
                    }}
                ></span>
            {/if}
        </span>
    {/each}
</div>

<style>
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
        width: 100%;
    }

    .head-cell {
        position: relative;
        display: flex;
        align-items: center;
        min-width: 0;
        overflow: visible;
        user-select: none;
    }

    .col-title {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        flex: 1;
    }

    .frames .col-title {
        text-align: right;
    }

    .col-resizer {
        position: absolute;
        right: -4px;
        top: 0;
        bottom: 0;
        width: 8px;
        cursor: col-resize;
        user-select: none;
        touch-action: none;
        z-index: 3;
    }

    .col-resizer::after {
        content: '';
        position: absolute;
        left: 3px;
        top: 2px;
        bottom: 2px;
        width: 2px;
        background-color: #e0e0e0;
        border-radius: 1px;
    }

    .col-resizer:hover::after,
    .col-resizer.active::after {
        background-color: #0066cc;
    }
</style>
