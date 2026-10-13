import { describe, expect, test } from '@jest/globals';
import { parseDSL, productions, applySensorFilter, type Suggestion } from '../dsl';
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

    test('parses handler', () => {
        let sx = parseDSL('handler eq "http"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    handler: 'http',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    handler: 'smb',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    handler: undefined,
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
    });

    test('parses transport', () => {
        let sx = parseDSL('transport eq "udp"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(53, '123', '192.168.1.1', undefined, 'Rule: UDP', {
                    transport: 'udp',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    transport: 'tcp',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
        // Events that never reported a transport cannot match one.
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    transport: undefined,
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
    });

    test('transport comparison ignores case', () => {
        let sx = parseDSL('transport eq "udp"');
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(53, '123', '192.168.1.1', undefined, 'Rule: UDP', {
                    transport: 'UDP',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
    });

    test('excludes a transport with ne', () => {
        let sx = parseDSL('transport ne "tcp"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(53, '123', '192.168.1.1', undefined, 'Rule: UDP', {
                    transport: 'udp',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    transport: 'tcp',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
    });

    test('combines transport with other clauses', () => {
        let sx = parseDSL('transport eq "udp" and udp.port == 53');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(53, '123', '192.168.1.1', undefined, 'Rule: UDP', {
                    transport: 'udp',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(53, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    transport: 'tcp',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
    });

    test('parses sensor eq and ==', () => {
        let sx = parseDSL('sensor eq "abcd1234"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'abcd1234',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'wxyz5678',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();

        let sx2 = parseDSL('sensor == "abcd1234"');
        expect(sx2.lexErrors).toHaveLength(0);
        expect(sx2.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'abcd1234',
                }),
                sx2.cst!,
            ),
        ).toBeTruthy();
    });

    test('parses sensor ne and !=', () => {
        let sx = parseDSL('sensor ne "abcd1234"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'wxyz5678',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'abcd1234',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();

        let sx2 = parseDSL('sensor != "abcd1234"');
        expect(sx2.lexErrors).toHaveLength(0);
        expect(sx2.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'wxyz5678',
                }),
                sx2.cst!,
            ),
        ).toBeTruthy();
    });

    test('matches sensor case-insensitively and handles prefix/UUID matching', () => {
        let sx = parseDSL('sensor eq "ABCD1234"');
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'abcd1234',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();

        // full UUID queried against 8-char event sensorID
        let sxUUID = parseDSL('sensor eq "abcd1234-5678-90ab-cdef-1234567890ab"');
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'abcd1234',
                }),
                sxUUID.cst!,
            ),
        ).toBeTruthy();

        // 8-char prefix queried against full UUID event sensorID
        let sxPrefix = parseDSL('sensor eq "abcd1234"');
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'abcd1234-5678-90ab-cdef-1234567890ab',
                }),
                sxPrefix.cst!,
            ),
        ).toBeTruthy();
    });

    test('combines sensor with other clauses', () => {
        let sx = parseDSL('sensor == "abcd1234" and tcp.port == 80');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'abcd1234',
                    transport: 'tcp',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'other123',
                    transport: 'tcp',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();

        let sxNot = parseDSL('not sensor == "abcd1234"');
        expect(sxNot.lexErrors).toHaveLength(0);
        expect(sxNot.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'other123',
                }),
                sxNot.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    sensorID: 'abcd1234',
                }),
                sxNot.cst!,
            ),
        ).toBeFalsy();
    });

    test('computes suggestions for sensor query', () => {
        let state1 = { suggestions: [] as string[], partialToken: null };
        parseDSL('', state1);
        expect(state1.suggestions).toContain('sensor');

        let state2 = { suggestions: [] as string[], partialToken: null };
        parseDSL('sensor ', state2);
        expect(state2.suggestions).toEqual(['eq', 'ne', '==', '!=']);

        let state3 = { suggestions: [] as string[], partialToken: null };
        parseDSL('sensor == ', state3);
        expect(state3.suggestions).toEqual(['"sensor-id"']);
    });

    test('computes suggestions for sensor query with known sensors', () => {
        let state = { suggestions: [] as Suggestion[], partialToken: null };
        parseDSL('sensor == ', state, {
            sensors: [{ id: 'abcd1234', name: 'alpha-sensor' }, { id: 'efgh5678' }],
        });
        expect(state.suggestions).toEqual([
            { value: '"abcd1234"', label: '"abcd1234" (alpha-sensor)' },
            { value: '"efgh5678"', label: '"efgh5678"' },
        ]);
    });

    test('filters sensor suggestions with partial query', () => {
        let state = { suggestions: [] as Suggestion[], partialToken: null };
        parseDSL('sensor == ab', state, {
            sensors: [
                { id: 'abcd1234', name: 'alpha-sensor' },
                { id: 'efgh5678', name: 'beta-sensor' },
            ],
        });
        expect(state.suggestions).toEqual([
            { value: '"abcd1234"', label: '"abcd1234" (alpha-sensor)' },
        ]);

        let stateWithQuote = { suggestions: [] as Suggestion[], partialToken: null };
        parseDSL('sensor == "ab', stateWithQuote, {
            sensors: [
                { id: 'abcd1234', name: 'alpha-sensor' },
                { id: 'efgh5678', name: 'beta-sensor' },
            ],
        });
        expect(stateWithQuote.suggestions).toEqual([
            { value: '"abcd1234"', label: '"abcd1234" (alpha-sensor)' },
        ]);

        let stateByName = { suggestions: [] as Suggestion[], partialToken: null };
        parseDSL('sensor == beta', stateByName, {
            sensors: [
                { id: 'abcd1234', name: 'alpha-sensor' },
                { id: 'efgh5678', name: 'beta-sensor' },
            ],
        });
        expect(stateByName.suggestions).toEqual([
            { value: '"efgh5678"', label: '"efgh5678" (beta-sensor)' },
        ]);

        let stateByNameQuote = { suggestions: [] as Suggestion[], partialToken: null };
        parseDSL('sensor == "beta', stateByNameQuote, {
            sensors: [
                { id: 'abcd1234', name: 'alpha-sensor' },
                { id: 'efgh5678', name: 'beta-sensor' },
            ],
        });
        expect(stateByNameQuote.suggestions).toEqual([
            { value: '"efgh5678"', label: '"efgh5678" (beta-sensor)' },
        ]);
    });

    test('applySensorFilter sets and updates filter correctly', () => {
        expect(applySensorFilter('', 'sensor-1')).toBe('sensor == "sensor-1"');
        expect(applySensorFilter('sensor == "sensor-1"', 'sensor-2')).toBe('sensor == "sensor-2"');
        expect(applySensorFilter('tcp.port == 80', 'sensor-3')).toBe(
            'tcp.port == 80 and sensor == "sensor-3"',
        );
        expect(applySensorFilter('tcp.port == 80 and sensor == "sensor-3"', 'sensor-4')).toBe(
            'tcp.port == 80 and sensor == "sensor-4"',
        );
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

    test('evaluates sensor', () => {
        let sx = parseDSL('sensor eq "sensorID"');
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(noRule(), sx.cst!)).toBeTruthy();
    });

    test('evaluates a compound query', () => {
        let sx = parseDSL('tcp.port == 445 and ip.src eq 203.0.113.10');
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(noRule(), sx.cst!)).toBeTruthy();
    });
});
