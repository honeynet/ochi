import { formatPort, type Event } from './event';
import { parseDSL } from './dsl';
import { filterEvent } from './eventFilter';
import type { QueryCstNode } from './generated/chevrotain_dts';
import type { Query } from './query';

// Statistics over the live event stream. They are kept in the browser only: counted as
// events arrive and saved to localStorage, so they survive a reload but not a new browser.

export const STATS_STORAGE_KEY = 'stats';
const STATS_VERSION = 1;

// Distinct ports and handlers counted one by one. Once a table is full, events with a new
// key go to OTHER_KEY instead, so a scan across every port can't grow storage without bound.
export const MAX_KEYS = 500;
export const OTHER_KEY = 'other';

export const RATE_WINDOW_MS = 60_000;

export interface Stats {
    since: string; // when counting started, ISO 8601
    total: number;
    ports: Map<string, number>; // keyed by formatPort, e.g. "tcp/445"
    handlers: Map<string, number>;
    queries: Map<string, number>; // keyed by saved query id
}

export function emptyStats(now: Date = new Date()): Stats {
    return {
        since: now.toISOString(),
        total: 0,
        ports: new Map(),
        handlers: new Map(),
        queries: new Map(),
    };
}

function bump(counts: Map<string, number>, key: string) {
    if (!counts.has(key) && counts.size >= MAX_KEYS) {
        key = OTHER_KEY;
    }
    counts.set(key, (counts.get(key) ?? 0) + 1);
}

export function countEvent(stats: Stats, event: Event, matchedQueryIds: string[] = []) {
    stats.total++;
    bump(stats.ports, formatPort(event));
    bump(stats.handlers, event.handler || 'unknown');
    for (const id of matchedQueryIds) {
        bump(stats.queries, id);
    }
}

// The n largest counts, ties in key order so the table doesn't reshuffle between renders.
export function topEntries(counts: Map<string, number>, n: number): [string, number][] {
    return [...counts].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).slice(0, n);
}

export function serializeStats(stats: Stats): string {
    return JSON.stringify({
        version: STATS_VERSION,
        since: stats.since,
        total: stats.total,
        ports: [...stats.ports],
        handlers: [...stats.handlers],
        queries: [...stats.queries],
    });
}

const isCount = (v: unknown): v is number => Number.isSafeInteger(v) && (v as number) >= 0;

function readCounts(value: unknown): Map<string, number> | undefined {
    if (!Array.isArray(value)) {
        return undefined;
    }
    const counts = new Map<string, number>();
    for (const entry of value) {
        if (!Array.isArray(entry) || typeof entry[0] !== 'string' || !isCount(entry[1])) {
            return undefined;
        }
        counts.set(entry[0], entry[1]);
    }
    return counts;
}

// Reads what serializeStats wrote. Anything else, from an older version or edited by hand,
// gives undefined so the caller can start over rather than show wrong numbers.
export function parseStats(text: string | null): Stats | undefined {
    if (!text) {
        return undefined;
    }
    let raw: any;
    try {
        raw = JSON.parse(text);
    } catch {
        return undefined;
    }
    if (
        typeof raw !== 'object' ||
        raw === null ||
        raw.version !== STATS_VERSION ||
        typeof raw.since !== 'string' ||
        isNaN(Date.parse(raw.since)) ||
        !isCount(raw.total)
    ) {
        return undefined;
    }
    const ports = readCounts(raw.ports);
    const handlers = readCounts(raw.handlers);
    const queries = readCounts(raw.queries);
    if (!ports || !handlers || !queries) {
        return undefined;
    }
    return { since: raw.since, total: raw.total, ports, handlers, queries };
}

// Counts the timestamps that fall within the last windowMs.
export class RateWindow {
    private times: number[] = [];

    constructor(private windowMs: number = RATE_WINDOW_MS) {}

    add(now: number) {
        this.times.push(now);
        this.prune(now);
    }

    count(now: number): number {
        this.prune(now);
        return this.times.length;
    }

    clear() {
        this.times = [];
    }

    private prune(now: number) {
        let expired = 0;
        while (expired < this.times.length && this.times[expired] <= now - this.windowMs) {
            expired++;
        }
        if (expired > 0) {
            this.times.splice(0, expired);
        }
    }
}

export interface CompiledQuery {
    id: string;
    cst: QueryCstNode;
}

// Parses the saved queries once, so each event isn't parsed against every query. Queries
// that don't parse can't match anything and are left out.
export function compileQueries(queries: Query[]): CompiledQuery[] {
    const compiled: CompiledQuery[] = [];
    for (const query of queries) {
        if (!query.id || !query.content) {
            continue;
        }
        const result = parseDSL(query.content);
        if (
            !result.cst ||
            result.lexErrors.length > 0 ||
            result.parseErrors.length > 0 ||
            result.regexErrors.length > 0
        ) {
            continue;
        }
        compiled.push({ id: query.id, cst: result.cst });
    }
    return compiled;
}

export function matchingQueries(event: Event, queries: CompiledQuery[]): string[] {
    return queries
        .filter((query) => {
            try {
                return filterEvent(event, query.cst);
            } catch {
                return false;
            }
        })
        .map((query) => query.id);
}
