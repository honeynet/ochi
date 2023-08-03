import { writable, Writable } from 'svelte/store';
import type { QueryCstNode } from './generated/chevrotain_dts';
import type { Event } from './event';
import { ENV_DEV } from './constants';

const storedIsAuthenticated = localStorage.getItem('isAuthenticated');
export const isAuthenticated = writable(storedIsAuthenticated === 'true');
isAuthenticated.subscribe((value) => {
    localStorage.setItem('isAuthenticated', value === true ? 'true' : 'false');
});

export const user = writable({});
export const maxNumberOfMessages = writable(50);

export const stringFilter: Writable<string> = writable('');
export const filterActive: Writable<boolean> = writable(false);

export const env: Writable<String> = writable(ENV_DEV);
export const parsedFilter: Writable<QueryCstNode | undefined> = writable(undefined);
export const currentEvent: Writable<Event | undefined> = writable(undefined);

const storedToken = localStorage.getItem('token');
export const token = writable(storedToken);
token.subscribe((value) => {
    localStorage.setItem('token', value === null ? '' : value);
});
