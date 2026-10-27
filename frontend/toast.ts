import { writable } from 'svelte/store';

export type ToastType = 'error' | 'success' | 'info';

export type Toast = {
    id: number;
    message: string;
    type: ToastType;
};

export const toasts = writable<Toast[]>([]);

let nextId = 0;

interface ActiveTimer {
    timeoutId: ReturnType<typeof setTimeout> | null;
    remainingMs: number;
    startedAt: number;
}

const timers = new Map<number, ActiveTimer>();

export const DEFAULT_TOAST_DURATION = 5000;

function startTimer(id: number, ms: number) {
    clearTimer(id);
    const timeoutId = setTimeout(() => {
        dismissToast(id);
    }, ms);
    timers.set(id, {
        timeoutId,
        remainingMs: ms,
        startedAt: Date.now(),
    });
}

function clearTimer(id: number) {
    const entry = timers.get(id);
    if (entry?.timeoutId) {
        clearTimeout(entry.timeoutId);
    }
    timers.delete(id);
}

export function pauseToast(id: number) {
    const entry = timers.get(id);
    if (!entry || entry.timeoutId === null) return;

    clearTimeout(entry.timeoutId);
    const elapsed = Date.now() - entry.startedAt;
    const remainingMs = Math.max(0, entry.remainingMs - elapsed);
    timers.set(id, {
        timeoutId: null,
        remainingMs,
        startedAt: Date.now(),
    });
}

export function resumeToast(id: number) {
    const entry = timers.get(id);
    if (!entry || entry.timeoutId !== null) return;

    if (entry.remainingMs > 0) {
        startTimer(id, entry.remainingMs);
    } else {
        dismissToast(id);
    }
}

export function dismissToast(id: number) {
    clearTimer(id);
    toasts.update((all) => all.filter((t) => t.id !== id));
}

export function showToast(
    message: string,
    type: ToastType = 'info',
    durationMs = DEFAULT_TOAST_DURATION,
): number {
    let id: number;
    let isExisting = false;

    toasts.update((all) => {
        const existing = all.find((t) => t.message === message && t.type === type);
        if (existing) {
            id = existing.id;
            isExisting = true;
            return all;
        }
        id = nextId++;
        return [...all, { id, message, type }];
    });

    const toastId = id!;

    if (isExisting) {
        const entry = timers.get(toastId);
        // If caller explicitly asks for 0, make it persistent
        if (durationMs === 0) {
            clearTimer(toastId);
        } else if (entry) {
            // If it was already paused (hovered/focused), reset remaining time without starting timeout
            if (entry.timeoutId === null) {
                entry.remainingMs = durationMs;
                entry.startedAt = Date.now();
            } else {
                startTimer(toastId, durationMs);
            }
        } else {
            // It was not persistent or newly given duration
            startTimer(toastId, durationMs);
        }
    } else if (durationMs > 0) {
        startTimer(toastId, durationMs);
    }

    return toastId;
}

export function showError(
    error: unknown,
    fallback = 'Something went wrong',
    durationMs = DEFAULT_TOAST_DURATION,
): number {
    const message = error instanceof Error ? error.message : typeof error === 'string' ? error : '';
    return showToast(message || fallback, 'error', durationMs);
}
