import { describe, expect, test } from '@jest/globals';
import { formatEventTime } from '../event';

describe('formatEventTime', () => {
    // Both dates are built in local time, so the expectations hold in any timezone.
    const now = new Date(2026, 0, 2, 15, 0, 0);

    test('shows the time for events from today', () => {
        const ts = new Date(2026, 0, 2, 9, 5, 7).toISOString();
        expect(formatEventTime(ts, now)).toBe('09:05:07');
    });

    test('shows the date for older events', () => {
        const ts = new Date(2025, 11, 31, 23, 59, 0).toISOString();
        expect(formatEventTime(ts, now)).toBe('Dec 31');
    });

    test('is empty when the time is missing, zero or invalid', () => {
        expect(formatEventTime(undefined, now)).toBe('');
        expect(formatEventTime('0001-01-01T00:00:00Z', now)).toBe('');
        expect(formatEventTime('not a time', now)).toBe('');
    });
});
