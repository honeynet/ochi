import { get, writable, type Writable } from 'svelte/store';
import type { Event } from './event';
import {
    compileQueries,
    countEvent,
    emptyStats,
    matchingQueries,
    parseStats,
    RateWindow,
    serializeStats,
    STATS_STORAGE_KEY,
    type CompiledQuery,
    type Stats,
} from './stats';
import { userQueries } from './store';

const SAVE_DELAY_MS = 2_000;

function loadStats(): Stats {
    try {
        return parseStats(localStorage.getItem(STATS_STORAGE_KEY)) ?? emptyStats();
    } catch {
        return emptyStats();
    }
}

function saveStats() {
    saveTimer = undefined;
    try {
        localStorage.setItem(STATS_STORAGE_KEY, serializeStats(get(stats)));
    } catch {
        // Storage full or blocked: the counts still work for this session.
    }
}

export const stats: Writable<Stats> = writable(loadStats());

const rate = new RateWindow();
let compiled: CompiledQuery[] = [];
let saveTimer: ReturnType<typeof setTimeout> | undefined;

userQueries.subscribe((queries) => {
    compiled = compileQueries(queries);
});

window.addEventListener('pagehide', () => {
    if (saveTimer !== undefined) {
        clearTimeout(saveTimer);
        saveStats();
    }
});

// Counts an event from the live stream. Writes to storage are batched, at most one every
// SAVE_DELAY_MS, instead of one per event.
export function recordEvent(event: Event, now: number = Date.now()) {
    const current = get(stats);
    countEvent(current, event, matchingQueries(event, compiled));
    rate.add(now);
    stats.set(current);
    if (saveTimer === undefined) {
        saveTimer = setTimeout(saveStats, SAVE_DELAY_MS);
    }
}

export function eventsLastMinute(now: number = Date.now()): number {
    return rate.count(now);
}

export function resetStats() {
    if (saveTimer !== undefined) {
        clearTimeout(saveTimer);
    }
    rate.clear();
    stats.set(emptyStats());
    saveStats();
}
