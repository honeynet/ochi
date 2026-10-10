import { describe, expect, test } from '@jest/globals';
import { parseDSL, productions } from '../dsl';
import { filterEvent } from '../eventFilter';
import { generateTestEvent } from '../util';

describe('parseDSL', () => {
    test('parses AND query', () => {
        let sx = parseDSL('tcp.port eq 23 and ip.src eq 1.1.1.1');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);

        // console.log(JSON.stringify(sx, null, 2));
        expect(sx.toString()).toBeTruthy();
        expect(filterEvent(generateTestEvent(23, '123', '1.1.1.1'), sx.cst!)).toBeTruthy();
    });

    test('parses ip.src ==', () => {
        // console.dir(parseDSL('tcp.port eq 23 and tcp.port eq 445'));
        let sx = parseDSL('ip.src eq 192.168.1.1 and tcp.port eq 445');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        // console.log(JSON.stringify(sx, null, 2));
        expect(sx.toString()).toBeTruthy();
        expect(filterEvent(generateTestEvent(445, '123', '192.168.1.1'), sx.cst!)).toBeTruthy();
    });

    test('parses single query with "ne port"', () => {
        let sx = parseDSL('tcp.port ne 23');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        // console.log(JSON.stringify(sx, null, 2));
        expect(sx.toString()).toBeTruthy();
        expect(filterEvent(generateTestEvent(445, '', '192.168.1.1'), sx.cst!)).toBeTruthy();
    });

    test('parses single query with "ne port"', () => {
        let sx = parseDSL('udp.port ne 8080');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(sx.toString()).toBeTruthy();
        expect(
            filterEvent(generateTestEvent(445, '', '192.168.1.1', '', 'Rule: UDP'), sx.cst!),
        ).toBeTruthy();
    });

    test('returns lexer error', () => {
        let sx = parseDSL('cdp.port ne 8080');
        expect(sx.lexErrors.length).toBeGreaterThan(0);
        expect(sx.parseErrors).toHaveLength(0);
    });

    test('returns parser error', () => {
        let sx = parseDSL('tcp.port ne tcp.port');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors.length).toBe(1);
    });

    test('does not parse consecutive queries', () => {
        let sx = parseDSL('tcp.port eq 23 tcp.port eq 445');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors.length).toBeGreaterThanOrEqual(1);
    });

    test('parsing payload', () => {
        let sx = parseDSL('payload contains "something"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        let payload = Buffer.from('something').toString('base64');
        expect(
            filterEvent(generateTestEvent(445, '123', '192.168.1.1', payload), sx.cst!),
        ).toBeTruthy();
    });

    test('parsing payload ne', () => {
        let sx = parseDSL('not payload contains "banana"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        let payload = Buffer.from('something').toString('base64');
        expect(
            filterEvent(generateTestEvent(445, '123', '192.168.1.1', payload), sx.cst!),
        ).toBeTruthy();
    });

    test('payload contains and tcp.port', () => {
        let sx = parseDSL('payload contains "something" and tcp.port == 445');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        let payload = Buffer.from('something').toString('base64');
        expect(
            filterEvent(generateTestEvent(445, '123', '192.168.1.1', payload), sx.cst!),
        ).toBeTruthy();
    });

    test('parsing payload ne and tcp.port eq', () => {
        let sx = parseDSL('not payload contains "banana" and tcp.port != 445');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        let payload = Buffer.from('something').toString('base64');
        expect(
            filterEvent(generateTestEvent(445, '123', '192.168.1.1', payload), sx.cst!),
        ).toBeFalsy();
    });

    test('parses ip.dst against dstHost', () => {
        let sx = parseDSL('ip.dst eq 198.51.100.8');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    dstHost: '198.51.100.8',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    dstHost: '10.0.0.1',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
        expect(filterEvent(generateTestEvent(80, '123', '192.168.1.1'), sx.cst!)).toBeFalsy();
    });

    test('parses tcp.port against transport rather than rule text', () => {
        let sx = parseDSL('tcp.port == 445');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        // A TCP session on 445 whose rule names the handler, not the protocol.
        expect(
            filterEvent(
                generateTestEvent(445, '123', '192.168.1.1', undefined, 'Rule: SMB', {
                    transport: 'tcp',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        // transport wins over the rule text when the two disagree.
        expect(
            filterEvent(
                generateTestEvent(445, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    transport: 'udp',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
    });

    test('parses end.reason', () => {
        let sx = parseDSL('end.reason eq "client_close"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    endReason: 'client_close',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    endReason: 'timeout',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
    });
});

// Glutton only sets `rule`/`ruleName` when one of its rules matched the connection,
// so an event that matched no rule reaches the filter without them. Clauses that do
// not talk about the protocol must still be evaluated for those events.
describe('filterEvent on events that matched no rule', () => {
    const noRule = () =>
        generateTestEvent(
            445,
            '54321',
            '203.0.113.10',
            Buffer.from('something').toString('base64'),
            undefined,
            {
                rule: undefined,
                transport: 'tcp',
                endReason: 'timeout',
            },
        );

    test('evaluates ip.src', () => {
        let sx = parseDSL('ip.src eq 203.0.113.10');
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(noRule(), sx.cst!)).toBeTruthy();
    });

    test('evaluates payload contains', () => {
        let sx = parseDSL('payload contains "something"');
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(noRule(), sx.cst!)).toBeTruthy();
    });

    test('evaluates end.reason', () => {
        let sx = parseDSL('end.reason eq "timeout"');
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(noRule(), sx.cst!)).toBeTruthy();
    });

    test('evaluates tcp.port using transport', () => {
        let sx = parseDSL('tcp.port == 445');
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(noRule(), sx.cst!)).toBeTruthy();
    });

    test('evaluates a compound query', () => {
        let sx = parseDSL('tcp.port == 445 and ip.src eq 203.0.113.10');
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(noRule(), sx.cst!)).toBeTruthy();
    });
});
