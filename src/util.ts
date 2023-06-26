import type { Event } from './event';
import { now } from 'svelte/internal';

/**
 * Debounces a callback to prevent calling it too many times.
 *
 * The wrapped function is called only after `delay` milliseconds after
 * the last call to this function. This wrapper is useful for debouncing
 * in UI widgets like text inputs.
 * @param callback Function to be called
 * @param delay Milliseconds to wait after last event before the function is called
 * @returns
 */
export function debounce<T extends (...args: any[]) => void>(callback: T, delay: number): T {
    let timeoutId = undefined;

    return <T>((...args: any[]): void => {
        clearTimeout(timeoutId);
        timeoutId = setTimeout(() => {
            callback(...args);
        }, delay);
    });
}

const ports = [80, 443, 22, 8080, 65345];
const handlers = ["http", "rdp", "", null]

/**
 * Generates a random event used for UI testing.
 * @returns test event
 */
export function generateRandomTestEvent(): Event {
    return {
        handler: handlers[Math.floor(Math.random() * handlers.length)],
        connKey: [2, 2],
        dstPort: ports[Math.floor(Math.random() * ports.length)],
        rule: 'Rule: TCP',
        scanner: 'censys',
        sensorID: 'sensorID',
        srcHost: '1.1.1.1',
        srcPort: '4321',
        timestamp: now().toString(),
        payload: 'dGVzdA==', // test,
        decoded: {"test": 123},
    };
}
