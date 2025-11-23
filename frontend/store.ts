import { writable, type Writable } from 'svelte/store';
import type { QueryCstNode } from './generated/chevrotain_dts';
import type { Event } from './event';
import type { Query } from './query';
import { ENV_DEV } from './constants';

const storedIsAuthenticated = localStorage.getItem('isAuthenticated');
export const isAuthenticated = writable(storedIsAuthenticated === 'true');
isAuthenticated.subscribe((value) => {
    localStorage.setItem('isAuthenticated', value === true ? 'true' : 'false');
});

export const user = writable({});
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

export const env: Writable<String> = writable(ENV_DEV);
export const parsedFilter: Writable<QueryCstNode | undefined> = writable(undefined);
export const currentEvent: Writable<Event | undefined> = writable(undefined);

const storedActiveFilterId = localStorage.getItem('activeFilterId');
export const activeFilterId: Writable<string | undefined> = writable(
    storedActiveFilterId || undefined,
);
activeFilterId.subscribe((value) => {
    localStorage.setItem('activeFilterId', value!);
});

const storedToken = localStorage.getItem('token');
export const token = writable(storedToken);
token.subscribe((value) => {
    localStorage.setItem('token', value === null ? '' : value);
});
