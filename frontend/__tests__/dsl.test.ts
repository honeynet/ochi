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
});
