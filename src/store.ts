import { writable, Writable } from 'svelte/store';
import type { QueryCstNode } from './generated/chevrotain_dts';
import type { Event } from './event';

const storedIsAuthenticated = localStorage.getItem('isAuthenticated');
export const isAuthenticated = writable(storedIsAuthenticated === 'true');
isAuthenticated.subscribe((value) => {
    localStorage.setItem('isAuthenticated', value === true ? 'true' : 'false');
});

export const user = writable({});
export const maxNumberOfMessages = writable(50);
export const parsedFilter: Writable<QueryCstNode | undefined> = writable(undefined);
export const currentEvent: Writable<Event | undefined> = writable(undefined);

const storedToken = localStorage.getItem('token');
export const token = writable(storedToken);
token.subscribe((value) => {
    localStorage.setItem('token', value === null ? '' : value);
});
