import { writable } from 'svelte/store';

export type Toast = {
    id: number;
    message: string;
};

export const toasts = writable<Toast[]>([]);

let nextId = 0;

export function dismissToast(id: number) {
    toasts.update((all) => all.filter((t) => t.id !== id));
}

export function showError(error: unknown, fallback = 'Something went wrong') {
    const message = error instanceof Error ? error.message : typeof error === 'string' ? error : '';
    toasts.update((all) => [...all, { id: nextId++, message: message || fallback }]);
}
