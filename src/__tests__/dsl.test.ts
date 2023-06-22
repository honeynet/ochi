import { describe, expect, test } from '@jest/globals';
import { parseDSL, productions } from '../dsl';

describe('parseDSL', () => {
    test('parses AND query', () => {
        let sx = parseDSL('tcp.port eq 23 and tcp.port eq 445');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);

        // console.log(JSON.stringify(sx, null, 2));
        expect(sx.toString()).toBeTruthy();
    });

    test('parses ip.src ==', () => {
        // console.dir(parseDSL('tcp.port eq 23 and tcp.port eq 445'));
        let sx = parseDSL('ip.src eq 192.168.1.1 and tcp.port eq 445');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        // console.log(JSON.stringify(sx, null, 2));
        expect(sx.toString()).toBeTruthy();
    });

    test('parses single query with "ne port"', () => {
        let sx = parseDSL('tcp.port ne 23');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        // console.log(JSON.stringify(sx, null, 2));
        expect(sx.toString()).toBeTruthy();
    });

    test('parses single query with "== port"', () => {
        let sx = parseDSL('udp.port ne 8080');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(sx.toString()).toBeTruthy();
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
});
