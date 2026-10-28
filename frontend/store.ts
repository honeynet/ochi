import { writable, derived, get, type Writable, type Readable } from 'svelte/store';
import type { QueryCstNode } from './generated/chevrotain_dts';
import type { Event } from './event';
import type { Query } from './query';
import type { Sensor } from './sensor';
import { applySensorFilter } from './dsl';
import { ENV_PROD } from './constants';

export type UserProfile = {
    email?: string;
    name?: string;
    [key: string]: unknown;
};

const storedIsAuthenticated = localStorage.getItem('isAuthenticated');
export const isAuthenticated = writable(storedIsAuthenticated === 'true');
isAuthenticated.subscribe((value) => {
    localStorage.setItem('isAuthenticated', value === true ? 'true' : 'false');
});

export const user: Writable<UserProfile | null> = writable(null);
// Permissions granted by the user's roles, as returned by /api/session. The
// server enforces them; the UI only uses them to decide what to show.
export const permissions: Writable<string[]> = writable([]);
export const maxNumberOfMessages = writable(50);

export const userQueries: Writable<Query[]> = writable([]);

const storedStringFilter = localStorage.getItem('stringFilter');
export const stringFilter: Writable<string> = writable(storedStringFilter || '');
stringFilter.subscribe((value) => {
    if (!value) {
        localStorage.removeItem('stringFilter');
    } else {
        localStorage.setItem('stringFilter', value);
    }
});

export const filterActive: Writable<boolean> = writable(false);

export const env: Writable<string> = writable(ENV_PROD);
export const parsedFilter: Writable<QueryCstNode | undefined> = writable(undefined);
export const currentEvent: Writable<Event | undefined> = writable(undefined);

const storedActiveFilterId = localStorage.getItem('activeFilterId');
export const activeFilterId: Writable<string | undefined> = writable(
    storedActiveFilterId || undefined,
);
activeFilterId.subscribe((value) => {
    if (!value) {
        localStorage.removeItem('activeFilterId');
    } else {
        localStorage.setItem('activeFilterId', value);
    }
});

const storedToken = localStorage.getItem('token') ?? '';
export const token: Writable<string> = writable(storedToken);
token.subscribe((value) => {
    if (!value) {
        localStorage.removeItem('token');
    } else {
        localStorage.setItem('token', value);
    }
});

export interface SensorOption {
    id: string;
    name?: string;
}

export const userSensors: Writable<Sensor[]> = writable([]);
export const observedSensorIds: Writable<string[]> = writable([]);

export const knownSensors: Readable<SensorOption[]> = derived(
    [userSensors, observedSensorIds],
    ([$userSensors, $observedIds]) => {
        const map = new Map<string, string | undefined>();
        for (const s of $userSensors) {
            map.set(s.id, s.name);
        }
        for (const id of $observedIds) {
            if (!map.has(id)) {
                map.set(id, undefined);
            }
        }
        const list: SensorOption[] = [];
        for (const [id, name] of map.entries()) {
            list.push({ id, name });
        }
        return list;
    },
);

export function recordObservedSensorId(id: string) {
    if (!id) return;
    observedSensorIds.update((ids) => {
        if (ids.includes(id)) return ids;
        return [...ids, id];
    });
}

export function filterBySensor(sensorID: string) {
    if (!sensorID) return;
    activeFilterId.set(undefined);
    stringFilter.update((current) => applySensorFilter(current, sensorID));
}

import {
    loadColumnWidths,
    hasSavedColumnWidths,
    saveColumnWidths,
    DEFAULT_COLUMN_WIDTHS,
    type EventColumnWidths,
} from './columnWidths';

export * from './columnWidths';

let isStoreInitialized = false;
export const hasCustomColumnWidths: Writable<boolean> = writable(hasSavedColumnWidths());
export const eventColumnWidths: Writable<EventColumnWidths> = writable(loadColumnWidths());
eventColumnWidths.subscribe((val) => {
    if (!isStoreInitialized) {
        isStoreInitialized = true;
        return;
    }
    saveColumnWidths(val);
    hasCustomColumnWidths.set(true);
});

export function resetColumnWidth(col?: keyof EventColumnWidths) {
    if (col) {
        eventColumnWidths.update((w) => ({
            ...w,
            [col]: DEFAULT_COLUMN_WIDTHS[col],
        }));
    } else {
        try {
            localStorage.removeItem('eventColumnWidths');
        } catch {}
        hasCustomColumnWidths.set(false);
        eventColumnWidths.set({ ...DEFAULT_COLUMN_WIDTHS });
    }
}
