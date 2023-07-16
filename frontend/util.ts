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
const handlers = ['http', 'rdp', '', null];

function generateRandomString(length: number): string {
    let result = '';
    const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    const charactersLength = characters.length;
    let counter = 0;
    while (counter < length) {
        result += characters.charAt(Math.floor(Math.random() * charactersLength));
        counter += 1;
    }
    return result;
}

function generateUUID() {
    // Public Domain/MIT
    var d = new Date().getTime(); //Timestamp
    var d2 =
        (typeof performance !== 'undefined' && performance.now && performance.now() * 1000) || 0; //Time in microseconds since page-load or 0 if unsupported
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        var r = Math.random() * 16; //random number between 0 and 16
        if (d > 0) {
            //Use timestamp until depleted
            r = (d + r) % 16 | 0;
            d = Math.floor(d / 16);
        } else {
            //Use microseconds since page-load if supported
            r = (d2 + r) % 16 | 0;
            d2 = Math.floor(d2 / 16);
        }
        return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
    });
}

/**
 * Generates a random event used for UI testing
 * @returns test event
 */
export function generateRandomTestEvent(): Event {
    let payload = `test ${generateRandomString(10 + Math.floor(Math.random() * 100))}`;
    return {
        handler: handlers[Math.floor(Math.random() * handlers.length)],
        connKey: [2, 2],
        dstPort: ports[Math.floor(Math.random() * ports.length)],
        rule: 'Rule: TCP',
        scanner: 'censys',
        sensorID: generateUUID(),
        srcHost: '1.1.1.1',
        srcPort: '4321',
        timestamp: now().toString(),
        payload: btoa(payload),
        decoded: { payload: payload },
    };
}

/**
 * Generates an event used for testing
 * @returns test event
 */
export function generateTestEvent(
    dport: number,
    sport?: string,
    sip?: string,
    payload?: string,
    rule: string = 'Rule: TCP',
): Event {
    return {
        handler: handlers[Math.floor(Math.random() * handlers.length)],
        connKey: [2, 2],
        dstPort: dport,
        rule: rule,
        scanner: 'censys',
        sensorID: 'sensorID',
        srcHost: sip,
        srcPort: sport,
        timestamp: now().toString(),
        payload: payload,
        decoded: { payload: 'test' },
    };
}
