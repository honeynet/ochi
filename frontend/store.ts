import { writable, type Writable } from 'svelte/store';
import type { QueryCstNode } from './generated/chevrotain_dts';
import type { Event } from './event';
import type { Query } from './query';
import { ENV_DEV } from './constants';

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

export const env: Writable<string> = writable(ENV_DEV);
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
