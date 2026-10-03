<script lang="ts">
    import type { Snippet } from 'svelte';

    const STACK_BREAKPOINT = 710;
    const MIN_RATIO = 0.2;
    const MAX_RATIO = 0.8;

    interface Props {
        left: Snippet;
        right: Snippet;
    }

    let { left, right }: Props = $props();

    let width = $state(0);
    let leftRatio = $state(0.2);
    let dragging = $state(false);

    let stacked = $derived(width > 0 && width < STACK_BREAKPOINT);
    let leftPercent = $derived(Math.round(leftRatio * 100));

    function clampRatio(value: number) {
        return Math.min(MAX_RATIO, Math.max(MIN_RATIO, value));
    }

    function splitElement(event: Event) {
        const handle = event.currentTarget;
        if (!(handle instanceof HTMLElement)) {
            return undefined;
        }
        const parent = handle.parentElement;
        return parent instanceof HTMLElement ? parent : undefined;
    }

    function ratioFromClientX(clientX: number, split: HTMLElement) {
        const rect = split.getBoundingClientRect();
        if (rect.width <= 0) {
            return leftRatio;
        }
        return clampRatio((clientX - rect.left) / rect.width);
    }

    function onPointerDown(event: PointerEvent) {
        if (stacked) {
            return;
        }
        const handle = event.currentTarget;
        const split = splitElement(event);
        if (!(handle instanceof HTMLElement) || !split) {
            return;
        }
        handle.setPointerCapture(event.pointerId);
        dragging = true;
        leftRatio = ratioFromClientX(event.clientX, split);
    }

    function onPointerMove(event: PointerEvent) {
        if (!dragging) {
            return;
        }
        const split = splitElement(event);
        if (!split) {
            return;
        }
        leftRatio = ratioFromClientX(event.clientX, split);
    }

    function onPointerUp(event: PointerEvent) {
        dragging = false;
        const handle = event.currentTarget;
        if (handle instanceof HTMLElement && handle.hasPointerCapture(event.pointerId)) {
            handle.releasePointerCapture(event.pointerId);
        }
    }

    function onKeyDown(event: KeyboardEvent) {
        if (stacked) {
            return;
        }
        const step = event.shiftKey ? 0.1 : 0.02;
        if (event.key === 'ArrowLeft' || event.key === 'ArrowDown') {
            event.preventDefault();
            leftRatio = clampRatio(leftRatio - step);
        } else if (event.key === 'ArrowRight' || event.key === 'ArrowUp') {
            event.preventDefault();
            leftRatio = clampRatio(leftRatio + step);
        }
    }
</script>

<div class={['split', { stacked }]} style:--left-ratio={leftRatio} bind:clientWidth={width}>
    <div class="pane pane-left">
        {@render left()}
    </div>
    {#if !stacked}
        <div
            class="handle"
            role="slider"
            aria-orientation="vertical"
            aria-valuenow={leftPercent}
            aria-valuemin={20}
            aria-valuemax={80}
            aria-label="Resize columns"
            tabindex="0"
            onpointerdown={onPointerDown}
            onpointermove={onPointerMove}
            onpointerup={onPointerUp}
            onpointercancel={onPointerUp}
            onkeydown={onKeyDown}
        ></div>
    {/if}
    <div class="pane pane-right">
        {@render right()}
    </div>
</div>

<style>
    .split {
        display: flex;
        flex: 1;
        min-width: 0;
        min-height: 0;
        width: 100%;
    }

    .stacked {
        flex-direction: column;
    }

    .pane {
        display: flex;
        flex-direction: column;
        min-width: 0;
        min-height: 0;
        overflow: auto;
    }

    .pane-left {
        flex: 0 0 calc(var(--left-ratio) * 100%);
    }

    .pane-right {
        flex: 1 1 0;
    }

    .stacked .pane-left,
    .stacked .pane-right {
        flex: 1 1 0;
    }

    .handle {
        flex: 0 0 6px;
        cursor: col-resize;
        touch-action: none;
        user-select: none;
        background: #e4e4e4;
        border-left: 1px solid #ccc;
        border-right: 1px solid #ccc;
    }

    .handle:hover,
    .handle:focus {
        background: #c8c8c8;
        outline: none;
    }
</style>
