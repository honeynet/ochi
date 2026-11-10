import type {
    QueryCstNode,
    BooleanClauseCstNode,
    BinaryOperatorCstChildren,
    BooleanSuffixClauseCstNode,
} from './generated/chevrotain_dts';
import type { Event } from './event';
import { parseStringLiteral, patternFor } from './dsl';

// One clause of a query chain, with the `not` that may precede it.
interface Term {
    clause: BooleanClauseCstNode;
    negate: boolean;
}

/**
 * Evaluates a query with `and` binding tighter than `or`, so `A and B or C` is
 * `(A and B) or C`, as in Wireshark display filters.
 *
 * The grammar nests a chain to the right (`A and (B or (C))`), so the chain is
 * flattened first and then evaluated as an `or` of `and` groups.
 */
export function filterEvent(event: Event, cst: QueryCstNode): boolean {
    const terms: Term[] = [];
    const operators: ('and' | 'or')[] = [];
    collectChain(cst, terms, operators);

    let groupResult = true;
    for (let i = 0; i < terms.length; i++) {
        if (i > 0 && operators[i - 1] === 'or') {
            if (groupResult) {
                return true;
            }
            groupResult = true;
        }
        if (groupResult) {
            const result = filterByBooleanClause(event, terms[i].clause);
            groupResult = terms[i].negate ? !result : result;
        }
    }
    return groupResult;
}

function collectChain(cst: QueryCstNode, terms: Term[], operators: ('and' | 'or')[]): void {
    let query: QueryCstNode | undefined = cst;
    while (query) {
        const children: QueryCstNode['children'] = query.children;
        const clause = children.booleanClause?.[0];
        const suffix: BooleanSuffixClauseCstNode | undefined = children.booleanSuffixClause?.[0];
        if (!clause || !suffix) {
            throw new Error(
                children.NOT ? 'Missing boolean clause for NOT query' : 'Unexpected node at query',
            );
        }
        terms.push({ clause, negate: !!children.NOT });

        const suffixChildren = suffix.children;
        if (suffixChildren.AND || suffixChildren.OR) {
            query = suffixChildren.query?.[0];
            if (!query) {
                throw new Error('Missing query for boolean suffix clause');
            }
            operators.push(suffixChildren.AND ? 'and' : 'or');
        } else {
            query = undefined;
        }
    }
}

/**
 * Decides whether an event was carried over the given transport protocol.
 *
 * `transport` is set on every event, so it is the authoritative source. The rule
 * text is only populated when a rule matched, so it is used as a fallback for
 * events from producers that do not report a transport.
 */
function transportMatches(event: Event, protocol: 'tcp' | 'udp'): boolean {
    if (event.transport) {
        return event.transport.toLowerCase() === protocol;
    }
    if (event.rule) {
        return event.rule.toLowerCase().includes(protocol);
    }
    throw new Error(`Missing transport and rule in event for ${protocol} port matching`);
}

function filterByBooleanClause(event: Event, booleanClauseCstNode: BooleanClauseCstNode): boolean {
    // A parenthesized group is evaluated as a query of its own
    const group = booleanClauseCstNode.children.query?.[0];
    if (group) {
        return filterEvent(event, group);
    }

    // We do not support unary clause yet, only binary clause is supported
    const binaryClause = booleanClauseCstNode.children.binaryClause?.[0];
    if (!binaryClause) {
        throw new Error('Missing binary clause for boolean clause');
    }

    const children = binaryClause.children;
    if (children.ipItemClause) {
        const ipv4Token = children.IPV4?.[0];
        if (!ipv4Token) {
            throw new Error('Missing IPv4 address for ip clause');
        }

        const binaryOperator = children.binaryOperator?.[0];
        if (!binaryOperator) {
            throw new Error('Missing binary operator for ip clause');
        }

        return equalityCheck(event.srcHost, ipv4Token.image, binaryOperator.children);
    } else if (children.stringItemClause) {
        const field = children.stringItemClause[0].children;
        const stringToken = children.STRING?.[0];
        const binaryOperator = children.binaryOperator?.[0];
        if (!stringToken) {
            throw new Error('Missing string for string clause');
        }
        if (!binaryOperator) {
            throw new Error('Missing binary operator for string clause');
        }
        const value = parseStringLiteral(stringToken.image);
        if (field.END_REASON) {
            return stringMatch(event.endReason, value, binaryOperator.children);
        } else if (field.HANDLER) {
            // a handler clause matches any layer of the stack (tls, http inside socks, ...)
            const layers = event.handlers?.length ? event.handlers : [event.handler ?? ''];
            return equalityCheck(layers.includes(value), true, binaryOperator.children);
        } else if (field.TRANSPORT) {
            // Compared case insensitively, as the port clauses already treat transport.
            return stringMatch(event.transport, value, binaryOperator.children, true);
        } else if (field.SCANNER) {
            // Scanner names (censys, shodan, ...) are case insensitive.
            return stringMatch(event.scanner, value, binaryOperator.children, true);
        }
        throw new Error('Unexpected field in string clause');
    } else if (children.portItemClause) {
        let portItemClause = children.portItemClause[0].children;
        const port = children.PORT?.[0];
        const binaryOperator = children.binaryOperator?.[0];
        if (!binaryOperator) {
            throw new Error('Missing binary operator for ip clause');
        }
        if (!port) {
            throw new Error('Missing port for port clause');
        }
        let portNumber = Number(port.image);
        if (portItemClause.TCP_PORT) {
            return (
                transportMatches(event, 'tcp') &&
                equalityCheck(event.dstPort, portNumber, binaryOperator.children)
            );
        } else if (portItemClause.UDP_PORT) {
            return (
                transportMatches(event, 'udp') &&
                equalityCheck(event.dstPort, portNumber, binaryOperator.children)
            );
        } else {
            throw new Error('Unexpected missing portItemClause');
        }
    } else if (children.searchClause) {
        const { STRING, searchOperator } = children.searchClause[0].children;
        if (!event.payload) {
            return false;
        }
        let bytes: string;
        try {
            bytes = atob(event.payload);
        } catch {
            return false;
        }
        const operator = searchOperator[0].children;
        if (operator.MATCHES || operator.MATCHES_SMB) {
            return patternFor(STRING[0]).test(payloadText(bytes));
        }
        return bytes.includes(parseStringLiteral(STRING[0].image));
    } else {
        throw new Error('Unexpected booleanClauseCstNode');
    }
}

// stringMatch compares an event field with the string from a clause. A field the event
// did not report counts as empty, so `ne` still matches such events.
function stringMatch(
    field: string | undefined,
    value: string,
    binaryOperator: BinaryOperatorCstChildren,
    ignoreCase = false,
): boolean {
    const actual = field ?? '';
    return ignoreCase
        ? equalityCheck(actual.toLowerCase(), value.toLowerCase(), binaryOperator)
        : equalityCheck(actual, value, binaryOperator);
}

// atob gives one character per byte, so a pattern tested against it would be matched against
// bytes rather than text: `.` would match half of a multi-byte character and a pattern such as
// "caf." could never match "café". The payload is decoded as UTF-8 first, and one that is not
// valid UTF-8 is matched as bytes, which is the best reading available for it.
//
// Only the start of a payload is tested. The pattern comes from whoever is at the keyboard,
// filterEvent runs on the UI thread for every event in the list, and a backtracking pattern
// such as (a+)+$ costs time that grows sharply with the length of the subject. The cap bounds
// that; it does not remove it, since a pattern can still be slow on a short subject.
const maxPatternBytes = 16 * 1024;

function payloadText(bytes: string): string {
    const head = truncateUtf8(bytes, maxPatternBytes);
    try {
        return decodeURIComponent(
            head.replace(/[\s\S]/g, (c) => '%' + c.charCodeAt(0).toString(16).padStart(2, '0')),
        );
    } catch {
        return head;
    }
}

// truncateUtf8 cuts at the limit without splitting a character in half, which would otherwise
// leave the whole payload undecodable: continuation bytes are 10xxxxxx, so stepping back over
// them lands on the start of the character the cut fell inside.
function truncateUtf8(bytes: string, limit: number): string {
    if (bytes.length <= limit) {
        return bytes;
    }
    let end = limit;
    while (end > 0 && (bytes.charCodeAt(end) & 0xc0) === 0x80) {
        end--;
    }
    return bytes.slice(0, end);
}

function equalityCheck<T>(
    first: T,
    second: T,
    binaryOperatorCstChildren: BinaryOperatorCstChildren,
): boolean {
    if (binaryOperatorCstChildren.EQUAL || binaryOperatorCstChildren.EQUAL_SMB) {
        return first === second;
    } else if (binaryOperatorCstChildren.NOT_EQUAL || binaryOperatorCstChildren.NOT_EQUAL_SMB) {
        return first !== second;
    }
    throw new Error('Unexpected binaryOperatorCstChildren');
}
