import type {
    QueryCstNode,
    BooleanClauseCstNode,
    BinaryOperatorCstChildren,
} from './generated/chevrotain_dts';
import type { Event } from './event';

export function filterEvent(event: Event, cst: QueryCstNode): boolean {
    let children = cst.children;
    if (children.booleanClause) {
        let booleanResult = filterByBooleanClause(event, children.booleanClause[0]);

        let booleanSuffixClauseC = children.booleanSuffixClause[0].children;

        if (booleanSuffixClauseC.AND) {
            return booleanResult && filterEvent(event, booleanSuffixClauseC.query[0]);
        } else if (booleanSuffixClauseC.OR) {
            return booleanResult || filterEvent(event, booleanSuffixClauseC.query[0]);
        } else {
            return booleanResult;
        }
    } else if (children.NOT) {
        return !filterEvent(event, cst.children.query[0]);
    } else {
        throw new Error('Unexpected node at query');
    }
}

function filterByBooleanClause(event: Event, booleanClauseCstNode: BooleanClauseCstNode): boolean {
    // We do not support unary clause yet, only binary clause is supported
    let children = booleanClauseCstNode.children.binaryClause[0].children;
    if (children.ipItemClause) {
        if (children.ipItemClause[0].children.IP_DST) {
            throw new Error('ip.dst is not supported yet');
        } else if (!children.ipItemClause[0].children.IP_SRC) {
            throw new Error('ip.src is missing');
        }
        return equalityCheck(
            event.srcHost,
            children.IPV4[0].image,
            children.binaryOperator[0].children,
        );
    } else if (children.portItemClause) {
        let portItemClause = children.portItemClause[0].children;
        let portNumber = Number(children.PORT[0].image);
        // TODO: proper protocol matching, for now checking only RULE
        if (portItemClause.TCP_PORT) {
            return (
                event.rule.toLowerCase().includes('tcp') &&
                equalityCheck(event.dstPort, portNumber, children.binaryOperator[0].children)
            );
        } else if (portItemClause.UDP_PORT) {
            return (
                event.rule.toLowerCase().includes('udp') &&
                equalityCheck(event.dstPort, portNumber, children.binaryOperator[0].children)
            );
        } else {
            throw new Error('Unexpected missing portItemClause');
        }
    } else if (children.searchClause) {
        let payloadString = children.searchClause[0].children.STRING[0].image.toLowerCase();
        let trimmedString = payloadString.substring(1, payloadString.length - 1);

        console.log(payloadString.length);
        return event.payload.toLowerCase().includes(trimmedString);
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
