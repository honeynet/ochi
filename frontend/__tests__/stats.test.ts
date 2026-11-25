import { describe, expect, test } from '@jest/globals';
import {
    compileQueries,
    countEvent,
    emptyStats,
    matchingQueries,
    MAX_KEYS,
    OTHER_KEY,
    parseStats,
    RateWindow,
    serializeStats,
    topEntries,
} from '../stats';
import { generateTestEvent } from '../util';

const event = (dstPort: number, handler: string, transport = 'tcp') =>
    generateTestEvent(dstPort, '1234', '203.0.113.10', '', 'Rule: TCP', { handler, transport });

describe('countEvent', () => {
    test('counts events by port, handler and matched query', () => {
        const s = emptyStats();
        countEvent(s, event(445, 'smb'), ['q1']);
        countEvent(s, event(445, 'smb'));
        countEvent(s, event(80, 'http'), ['q1', 'q2']);

        expect(s.total).toBe(3);
        expect(s.ports).toEqual(
            new Map([
                ['tcp/445', 2],
                ['tcp/80', 1],
            ]),
        );
        expect(s.handlers).toEqual(
            new Map([
                ['smb', 2],
                ['http', 1],
            ]),
        );
        expect(s.queries).toEqual(
            new Map([
                ['q1', 2],
                ['q2', 1],
            ]),
        );
    });

    test('counts events without a handler as unknown', () => {
        const s = emptyStats();
        countEvent(s, event(22, ''));
        expect(s.handlers.get('unknown')).toBe(1);
    });

    test('puts new keys past the cap into other', () => {
        const s = emptyStats();
        for (let port = 1; port <= MAX_KEYS; port++) {
            countEvent(s, event(port, 'tcp'));
        }
        countEvent(s, event(60000, 'tcp'));
        countEvent(s, event(60001, 'tcp'));
        // a port that is already counted keeps its own entry
        countEvent(s, event(1, 'tcp'));

        expect(s.ports.size).toBe(MAX_KEYS + 1);
        expect(s.ports.get(OTHER_KEY)).toBe(2);
        expect(s.ports.get('tcp/1')).toBe(2);
        expect(s.ports.has('tcp/60000')).toBe(false);
    });
});

describe('topEntries', () => {
    test('returns the largest counts first, ties by key', () => {
        const counts = new Map([
            ['b', 2],
            ['a', 2],
            ['c', 5],
            ['d', 1],
        ]);
        expect(topEntries(counts, 3)).toEqual([
            ['c', 5],
            ['a', 2],
            ['b', 2],
        ]);
    });
});

describe('serializeStats and parseStats', () => {
    test('round trip', () => {
        const s = emptyStats(new Date('2026-01-02T03:04:05Z'));
        countEvent(s, event(445, 'smb'), ['q1']);
        expect(parseStats(serializeStats(s))).toEqual(s);
    });

    test('reject anything else', () => {
        const valid = JSON.parse(serializeStats(emptyStats()));
        for (const text of [
            null,
            '',
            'not json',
            'null',
            '[]',
            JSON.stringify({ ...valid, version: 2 }),
            JSON.stringify({ ...valid, since: 'yesterday' }),
            JSON.stringify({ ...valid, total: -1 }),
            JSON.stringify({ ...valid, total: 1.5 }),
            JSON.stringify({ ...valid, ports: {} }),
            JSON.stringify({ ...valid, ports: [['tcp/80', '3']] }),
            JSON.stringify({ ...valid, handlers: [[1, 3]] }),
            JSON.stringify({ ...valid, queries: undefined }),
        ]) {
            expect(parseStats(text)).toBeUndefined();
        }
    });
});

describe('RateWindow', () => {
    test('counts only the last window', () => {
        const rate = new RateWindow(1000);
        rate.add(0);
        rate.add(500);
        rate.add(900);
        expect(rate.count(900)).toBe(3);
        expect(rate.count(1000)).toBe(2);
        expect(rate.count(1500)).toBe(1);
        expect(rate.count(1900)).toBe(0);
    });
});

describe('saved queries', () => {
    test('match events, and broken queries are left out', () => {
        const compiled = compileQueries([
            { id: 'smb', content: 'tcp.port eq 445' },
            { id: 'web', content: 'tcp.port eq 80 or tcp.port eq 443' },
            { id: 'broken', content: 'tcp.port eq' },
            { id: 'empty', content: '' },
            { content: 'tcp.port eq 445' },
        ]);

        expect(compiled.map((q) => q.id)).toEqual(['smb', 'web']);
        expect(matchingQueries(event(445, 'smb'), compiled)).toEqual(['smb']);
        expect(matchingQueries(event(443, 'http'), compiled)).toEqual(['web']);
        expect(matchingQueries(event(22, 'ssh'), compiled)).toEqual([]);
    });
});
