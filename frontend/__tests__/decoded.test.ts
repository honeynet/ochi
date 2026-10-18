import { describe, expect, test } from '@jest/globals';
import {
    countFrames,
    formatFrameCount,
    frameCountTitle,
    frameTableColumns,
    formatFrameValue,
    nestedFrames,
} from '../decoded';
import { formatDest, formatDuration, formatPort, hasTime } from '../event';
import { generateTestEvent } from '../util';

describe('frameTableColumns', () => {
    test('shows handler-specific keys for handlers without a column list', () => {
        const frames = [
            {
                direction: 'read',
                command: 'ClientHello',
                server_name: 'a.example',
                payload: 'AA==',
            },
            { direction: 'write', status: 'ok', cookie_present: true },
        ];
        expect(frameTableColumns(frames, 'dtls')).toEqual([
            'direction',
            'command',
            'status',
            'server_name',
            'cookie_present',
        ]);
    });

    test('orders listed handler columns first and keeps header', () => {
        const frames = [
            { direction: 'read', header: { tid: 1 }, native_os: 'Windows', account: 'x' },
        ];
        expect(frameTableColumns(frames, 'smb')).toEqual([
            'direction',
            'header',
            'account',
            'native_os',
        ]);
    });

    test('http shows per-frame ports', () => {
        const frames = [{ direction: 'read', command: 'GET', dest_port: 8080, src_port: 1234 }];
        expect(frameTableColumns(frames, 'http')).toEqual([
            'direction',
            'command',
            'dest_port',
            'src_port',
        ]);
    });
});

describe('formatFrameValue', () => {
    test('joins ID lists instead of hex-concatenating them', () => {
        expect(formatFrameValue([49195, 4865], 'cipher_suites')).toBe('49195, 4865');
        expect(formatFrameValue([18, 17, 23], 'etypes')).toBe('18, 17, 23');
        expect(formatFrameValue([0, -239], 'encodings')).toBe('0, -239');
    });

    test('hex-encodes known fixed-size byte arrays', () => {
        expect(formatFrameValue([0, 1, 255], 'info_hash')).toBe('0x0001ff');
    });

    test('formats headers', () => {
        expect(formatFrameValue({ tid: 1, uid: 2 }, 'header')).toBe('tid=1 uid=2');
    });
});

describe('event formatting', () => {
    test('formatPort includes transport', () => {
        const event = generateTestEvent(5060, undefined, undefined, undefined, 'Rule: UDP', {
            transport: 'udp',
        });
        expect(formatPort(event)).toBe('udp/5060');
        expect(formatDest({ ...event, dstHost: '192.0.2.1' })).toBe('192.0.2.1 udp/5060');
        expect(formatPort({ ...event, transport: undefined })).toBe(':5060');
    });

    test('formatDuration', () => {
        expect(formatDuration(250)).toBe('250ms');
        expect(formatDuration(32_400)).toBe('32.4s');
        expect(formatDuration(723_000)).toBe('12m03s');
        expect(formatDuration(3_723_000)).toBe('1h02m03s');
    });

    test('hasTime ignores Go zero time', () => {
        expect(hasTime('0001-01-01T00:00:00Z')).toBe(false);
        expect(hasTime(undefined)).toBe(false);
        expect(hasTime('2026-05-15T12:00:00Z')).toBe(true);
    });
});

describe('formatFrameCount', () => {
    test('splits decoded frames into received/sent', () => {
        const frames = [{ direction: 'read' }, { direction: 'write' }, { direction: 'read' }];
        expect(formatFrameCount(frames, 3)).toBe('2/1');
    });

    test('falls back to the total without directional frames', () => {
        expect(formatFrameCount(undefined, 4)).toBe('4');
        expect(formatFrameCount({ foo: 'bar' }, 1)).toBe('1');
        expect(formatFrameCount([{ command: 'GET' }], 1)).toBe('1');
        expect(formatFrameCount(undefined)).toBe('');
    });
});

describe('nested frames', () => {
    const socks = [
        { direction: 'read', command: 'socks4-connect' },
        { direction: 'write', command: 'socks4-connect', status: 'granted' },
        {
            direction: 'read',
            command: 'tunnel',
            requests: 1,
            http: [
                { direction: 'read', command: 'GET', path: '/ip', payload: 'AA==' },
                { direction: 'write', status: '200' },
            ],
        },
    ];

    test('nestedFrames finds embedded frame arrays only', () => {
        expect(nestedFrames(socks[2]).map((n) => [n.key, n.frames.length])).toEqual([['http', 2]]);
        expect(nestedFrames({ direction: 'read', methods: [0, 2], http: [] })).toEqual([]);
        expect(nestedFrames({ direction: 'read', args: ['a', 'b'] })).toEqual([]);
    });

    test('the nested key is not a column of the parent table', () => {
        expect(frameTableColumns(socks, 'socks')).toEqual([
            'direction',
            'command',
            'status',
            'requests',
        ]);
    });

    test('counts include nested frames', () => {
        expect(countFrames(socks)).toEqual({ received: 3, sent: 2, nested: 2 });
        expect(formatFrameCount(socks, 3)).toBe('3/2');
        expect(frameCountTitle(socks)).toBe('received/sent, including 2 nested');
    });

    test('flat frames keep the plain title', () => {
        expect(frameCountTitle([{ direction: 'read' }])).toBe('received/sent');
        expect(frameCountTitle(undefined)).toBe('received/sent');
    });
});
