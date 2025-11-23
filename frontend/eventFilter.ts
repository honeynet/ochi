import type {
    QueryCstNode,
    BooleanClauseCstNode,
    BinaryOperatorCstChildren,
    BooleanSuffixClauseCstNode,
} from './generated/chevrotain_dts';
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

function filterByBooleanClause(event: Event, booleanClauseCstNode: BooleanClauseCstNode): boolean {
    // We do not support unary clause yet, only binary clause is supported
    const binaryClause = booleanClauseCstNode.children.binaryClause?.[0];
    if (!binaryClause) {
        throw new Error('Missing binary clause for boolean clause');
    }

    const rule = event.rule ? event.rule.toLowerCase() : '';
    if (!rule) {
        throw new Error('Missing rule in event for port matching');
    }

    const children = binaryClause.children;
    if (children.ipItemClause) {
        if (children.ipItemClause[0].children.IP_DST) {
            throw new Error('ip.dst is not supported yet');
        } else if (!children.ipItemClause[0].children.IP_SRC) {
            throw new Error('ip.src is missing');
        }

        const ipv4Token = children.IPV4?.[0];
        if (!ipv4Token) {
            throw new Error('Missing IPv4 address for ip clause');
        }

        const binaryOperator = children.binaryOperator?.[0];
        if (!binaryOperator) {
            throw new Error('Missing binary operator for ip clause');
        }

        return equalityCheck(event.srcHost, ipv4Token.image, binaryOperator.children);
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
        // TODO: proper protocol matching, for now checking only RULE
        if (portItemClause.TCP_PORT) {
            if (!event.rule) {
                throw new Error('Missing rule in event for TCP port matching');
            }
            return (
                event.rule.toLowerCase().includes('tcp') &&
                equalityCheck(event.dstPort, portNumber, binaryOperator.children)
            );
        } else if (portItemClause.UDP_PORT) {
            return (
                rule.includes('udp') &&
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
        const payloadString = stringToken.image;
        const trimmedString = payloadString.substring(1, payloadString.length - 1);
        if (!event.payload) {
            return false;
        }
        return atob(event.payload).includes(trimmedString);
    } else {
        throw new Error('Unexpected booleanClauseCstNode');
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
