import { createToken, CstParser, Rule, Lexer, EMPTY_ALT } from 'chevrotain';

import type { ILexingError, IRecognitionException, IToken, TokenType } from 'chevrotain';
import type { QueryCstNode } from './generated/chevrotain_dts';

// Comparison
const eq = createToken({ name: 'EQUAL', pattern: /eq/, label: 'eq' });
const ne = createToken({ name: 'NOT_EQUAL', pattern: /ne/, label: 'ne' });
const eqSmb = createToken({ name: 'EQUAL_SMB', pattern: /==/, label: '==' });
const neSmb = createToken({ name: 'NOT_EQUAL_SMB', pattern: /!=/, label: '!=' });

// Boolean logic
const and = createToken({ name: 'AND', pattern: /and/, label: 'and' });
const or = createToken({ name: 'OR', pattern: /or/, label: 'or' });
const not = createToken({ name: 'NOT', pattern: /not/, label: 'not' });

// Search and match operators
const contains = createToken({ name: 'CONTAINS', pattern: /contains/, label: 'contains' });
const matches = createToken({ name: 'MATCHES', pattern: /matches/, label: 'matches' });
const matchesSmb = createToken({ name: 'MATCHES_SMB', pattern: /~/, label: '~' });

// Literals
const port = createToken({ name: 'PORT', pattern: /(?:0|[1-9]\d*)/, label: '<PORT>' });
const ipv4 = createToken({
    name: 'IPV4',
    pattern:
        /([01]?[0-9]?[0-9]|2[0-4][0-9]|25[0-5])\.([01]?[0-9]?[0-9]|2[0-4][0-9]|25[0-5])\.([01]?[0-9]?[0-9]|2[0-4][0-9]|25[0-5])\.([01]?[0-9]?[0-9]|2[0-4][0-9]|25[0-5])/,
    label: '<IP>',
});

const ipSrc = createToken({ name: 'IP_SRC', pattern: /ip\.src/, label: 'ip.src' });
const ipDst = createToken({ name: 'IP_DST', pattern: /ip\.dst/, label: 'ip.dst' });

const tcpPort = createToken({ name: 'TCP_PORT', pattern: /tcp\.port/, label: 'tcp.port' });
const udpPort = createToken({ name: 'UDP_PORT', pattern: /udp\.port/, label: 'udp.port' });

const payload = createToken({ name: 'PAYLOAD', pattern: /payload/, label: 'payload' });
const string = createToken({ name: 'STRING', pattern: /\"[a-zA-Z0-9]+\"/, label: '"msg"' });

const partial = createToken({
    name: 'PARTIAL',
    pattern: /[a-zA-Z][a-zA-Z0-9]*|\?/,
    label: 'partial',
});

const whiteSpace = createToken({
    name: 'WhiteSpace',
    pattern: /\s+/,
    group: Lexer.SKIPPED,
});

let allTokens = [
    whiteSpace,
    eq,
    ne,
    eqSmb,
    neSmb,

    and,
    or,
    not,

    contains,
    matches,
    matchesSmb,

    ipv4,
    port,

    ipSrc,
    ipDst,
    tcpPort,
    udpPort,

    payload,
    string,

    partial,
];

let queryLexer = new Lexer(allTokens);

class QueryParser extends CstParser {
    constructor() {
        super(allTokens);
        this.performSelfAnalysis();
    }

    public query = this.RULE('query', () => {
        this.OR([
            {
                ALT: () => {
                    this.SUBRULE(this.booleanClause);
                    this.SUBRULE(this.booleanSuffixClause);
                },
            },
            {
                ALT: () => {
                    this.CONSUME(not);
                    this.SUBRULE1(this.booleanClause);
                    this.SUBRULE1(this.booleanSuffixClause);
                },
            },
        ]);
    });

    private booleanSuffixClause = this.RULE('booleanSuffixClause', () => {
        this.OR([
            {
                ALT: () => {
                    this.CONSUME(or);
                    this.SUBRULE(this.query);
                },
            },
            {
                ALT: () => {
                    this.CONSUME(and);
                    this.SUBRULE1(this.query);
                },
            },
            {
                ALT: EMPTY_ALT('empty field'),
            },
        ]);
    });

    private booleanClause = this.RULE('booleanClause', () => {
        this.OR([
            {
                ALT: () => {
                    this.SUBRULE(this.binaryClause);
                },
            },
            // {
            //     ALT: () => {
            //         this.SUBRULE(this.unaryClause);
            //     }
            // },
        ]);
    });

    private searchClause = this.RULE('searchClause', () => {
        this.OR([
            {
                ALT: () => {
                    this.CONSUME(payload);
                    this.CONSUME(contains);
                    this.CONSUME(string);
                },
            },
        ]);
    });

    private portItemClause = this.RULE('portItemClause', () => {
        this.OR([
            {
                ALT: () => {
                    this.CONSUME(tcpPort);
                },
            },
            {
                ALT: () => {
                    this.CONSUME(udpPort);
                },
            },
        ]);
    });

    private ipItemClause = this.RULE('ipItemClause', () => {
        this.OR([
            {
                ALT: () => {
                    this.CONSUME(ipSrc);
                },
            },
            {
                ALT: () => {
                    this.CONSUME(ipDst);
                },
            },
        ]);
    });

    private binaryClause = this.RULE('binaryClause', () => {
        this.OR([
            {
                ALT: () => {
                    this.SUBRULE(this.portItemClause);
                    this.SUBRULE(this.binaryOperator);
                    this.CONSUME(port);
                },
            },
            {
                ALT: () => {
                    this.SUBRULE(this.ipItemClause);
                    this.SUBRULE1(this.binaryOperator);
                    this.CONSUME(ipv4);
                },
            },
            {
                ALT: () => {
                    this.SUBRULE(this.searchClause);
                },
            },
        ]);
    });

    private binaryOperator = this.RULE('binaryOperator', () => {
        this.OR([
            { ALT: () => this.CONSUME(eq) },
            { ALT: () => this.CONSUME(ne) },
            { ALT: () => this.CONSUME(eqSmb) },
            { ALT: () => this.CONSUME(neSmb) },
        ]);
    });
}

const parser = new QueryParser();

export interface PartialToken {
    image: string;
    startOffset: number;
}

export interface FilterState {
    suggestions: string[];
    partialToken: PartialToken | null;
}

export interface ParseResult {
    cst?: QueryCstNode;
    lexErrors: ILexingError[];
    parseErrors: IRecognitionException[];
}

export const productions: Record<string, Rule> = parser.getGAstProductions();

// create the HTML Text
export const serializedGrammar = parser.getSerializedGastProductions();

const QUERY_START_SUGGESTIONS = ['tcp.port', 'udp.port', 'ip.src', 'ip.dst', 'payload', 'not'];

const FIELD_SUGGESTIONS = ['tcp.port', 'udp.port', 'ip.src', 'ip.dst', 'payload'];

const OPERATOR_SUGGESTIONS = ['eq', 'ne', '==', '!='];
const BOOLEAN_SUFFIX_SUGGESTIONS = ['and', 'or'];

function tokenMatches(token: IToken, ...types: TokenType[]): boolean {
    return types.includes(token.tokenType);
}

function computeSuggestions(assistTokens: IToken[]): string[] {
    if (assistTokens.length === 0) {
        return QUERY_START_SUGGESTIONS;
    }

    const lastToken = assistTokens[assistTokens.length - 1];

    if (tokenMatches(lastToken, and, or)) {
        return QUERY_START_SUGGESTIONS;
    }

    if (tokenMatches(lastToken, not)) {
        return FIELD_SUGGESTIONS;
    }

    if (tokenMatches(lastToken, tcpPort, udpPort, ipSrc, ipDst)) {
        return OPERATOR_SUGGESTIONS;
    }

    if (tokenMatches(lastToken, eq, ne, eqSmb, neSmb)) {
        const fieldToken = assistTokens[assistTokens.length - 2];
        if (fieldToken && tokenMatches(fieldToken, tcpPort, udpPort)) {
            return ['<PORT>'];
        }
        if (fieldToken && tokenMatches(fieldToken, ipSrc, ipDst)) {
            return ['<IP>'];
        }
        return [];
    }

    if (tokenMatches(lastToken, port, ipv4, string)) {
        return BOOLEAN_SUFFIX_SUGGESTIONS;
    }

    if (tokenMatches(lastToken, payload)) {
        return ['contains'];
    }

    if (tokenMatches(lastToken, contains)) {
        return ['"msg"'];
    }

    return [];
}

export function parseDSL(
    text: string,
    filterState: FilterState = { suggestions: [], partialToken: null },
): ParseResult {
    const lexResult = queryLexer.tokenize(text);

    if (lexResult.errors.length > 0) {
        return {
            lexErrors: lexResult.errors,
            parseErrors: [],
        };
    }
    // setting a new input will RESET the parser instance's state.

    const assistTokens = lexResult.tokens.slice();
    let lastToken: IToken | undefined;
    let partialSuggestion = false;

    if (assistTokens.length > 0 && assistTokens[assistTokens.length - 1].tokenType === partial) {
        lastToken = assistTokens.pop();
        partialSuggestion = true;
    }

    let suggestions = computeSuggestions(assistTokens);
    if (partialSuggestion && lastToken && lastToken.image !== '?') {
        const partialImage = lastToken.image;
        suggestions = suggestions.filter((suggestion) => suggestion.startsWith(partialImage));
    }

    filterState.suggestions = suggestions;
    filterState.partialToken =
        partialSuggestion && lastToken
            ? { image: lastToken.image, startOffset: lastToken.startOffset }
            : null;

    parser.input = lexResult.tokens;
    // any top level rule may be used as an entry point
    const cst = parser.query();
    console.log(`the cst is ${JSON.stringify(cst)}`);

    return {
        // This is a pure grammar, the value will be undefined until we add embedded actions
        // or enable automatic CST creation.
        cst: cst as QueryCstNode,
        lexErrors: lexResult.errors,
        parseErrors: parser.errors,
    };
}
