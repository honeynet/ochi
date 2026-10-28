import type {
    QueryCstNode,
    BooleanClauseCstNode,
    BinaryOperatorCstChildren,
    BooleanSuffixClauseCstNode,
} from './generated/chevrotain_dts';
import type { IToken } from 'chevrotain';
import type { Event } from './event';

export function filterEvent(event: Event, cst: QueryCstNode): boolean {
    let children = cst.children;
    if (children.NOT) {
        if (!children.booleanClause || !children.booleanSuffixClause) {
            throw new Error('Missing boolean clause for NOT query');
        }
        return filterByBooleanClauseWithSuffix(
            event,
            children.booleanClause[0],
            children.booleanSuffixClause[0],
            true,
        );
    } else if (children.booleanClause && children.booleanSuffixClause) {
        return filterByBooleanClauseWithSuffix(
            event,
            children.booleanClause[0],
            children.booleanSuffixClause[0],
        );
    } else {
        throw new Error('Unexpected node at query');
    }
}

function filterByBooleanClauseWithSuffix(
    event: Event,
    booleanClause: BooleanClauseCstNode,
    suffix: BooleanSuffixClauseCstNode,
    negate: boolean = false,
): boolean {
    let booleanResult = filterByBooleanClause(event, booleanClause);
    if (negate) {
        booleanResult = !booleanResult;
    }

    let booleanSuffixClauseC = suffix.children;

    if (booleanSuffixClauseC.AND) {
        const nextQuery = booleanSuffixClauseC.query?.[0];
        if (!nextQuery) {
            throw new Error('Missing query for boolean suffix clause');
        }
        return booleanResult && filterEvent(event, nextQuery);
    } else if (booleanSuffixClauseC.OR) {
        const nextQuery = booleanSuffixClauseC.query?.[0];
        if (!nextQuery) {
            throw new Error('Missing query for boolean suffix clause');
        }
        return booleanResult || filterEvent(event, nextQuery);
    } else {
        return booleanResult;
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
        const searchClause = children.searchClause[0];
        const stringToken = searchClause.children.STRING?.[0];
        if (!stringToken) {
            throw new Error('Missing string token for search clause');
        }
        if (!event.payload) {
            return false;
        }
        let bytes: string;
        try {
            bytes = atob(event.payload);
        } catch {
            return false;
        }
        // A payload is bytes, and a term can be meant either way round: "\u0090\u0090" is two
        // raw bytes of a NOP sled, while "café" is text the payload carries as UTF-8. Both
        // readings are tried, so gaining the second does not give up the first.
        const term = searchTerm(stringToken);
        return bytes.includes(term.raw) || (term.utf8 !== term.raw && bytes.includes(term.utf8));
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

// Both readings of a search term, kept together. filterEvent runs for every event in the list,
// and again for every stored message whenever the filter changes, so a term is converted once
// per filter rather than once per event. A token lives exactly as long as the parsed filter
// that holds it, so the entry goes away with it.
const searchTerms = new WeakMap<IToken, { raw: string; utf8: string }>();

function searchTerm(token: IToken): { raw: string; utf8: string } {
    let term = searchTerms.get(token);
    if (!term) {
        const raw = parseStringLiteral(token.image);
        term = { raw, utf8: utf8ByteString(raw) };
        searchTerms.set(token, term);
    }
    return term;
}

// atob() yields one character per byte, so a term meant as text has to be turned into its UTF-8
// bytes before comparing; otherwise a term like "café" could never match a UTF-8 payload. ASCII
// terms come back unchanged. This is what unescape(encodeURIComponent(s)) does, written out
// because unescape is deprecated.
function utf8ByteString(s: string): string {
    // A lone surrogate is not text any encoder can represent, and encodeURIComponent rejects it
    // outright. UTF-8 encoders substitute U+FFFD for one, so do the same first: the term then
    // encodes like any other, instead of falling back to a value that, holding a code unit
    // above U+00FF, could never match a byte string anyway.
    const text = s.replace(/\p{Surrogate}/gu, '\uFFFD');
    return encodeURIComponent(text).replace(/%([0-9A-F]{2})/g, (_, hex) =>
        String.fromCharCode(parseInt(hex, 16)),
    );
}

export function parseStringLiteral(image: string): string {
    const raw = image.slice(1, -1);
    try {
        return JSON.parse(image);
    } catch {
        return raw.replace(/\\(.)/g, '$1');
    }
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
