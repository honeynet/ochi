import { createToken, CstParser, Rule, Lexer, EMPTY_ALT } from 'chevrotain';

import type { CstNode, ILexingError, IRecognitionException, IToken, TokenType } from 'chevrotain';
import type { QueryCstNode, SearchClauseCstChildren } from './generated/chevrotain_dts';

// Comparison
const eq = createToken({ name: 'EQUAL', pattern: /eq/, label: 'eq' });
const ne = createToken({ name: 'NOT_EQUAL', pattern: /ne/, label: 'ne' });
const eqSmb = createToken({ name: 'EQUAL_SMB', pattern: /==/, label: '==' });
const neSmb = createToken({ name: 'NOT_EQUAL_SMB', pattern: /!=/, label: '!=' });

// Boolean logic
const and = createToken({ name: 'AND', pattern: /and/, label: 'and' });
const or = createToken({ name: 'OR', pattern: /or/, label: 'or' });
const not = createToken({ name: 'NOT', pattern: /not/, label: 'not' });

// Grouping
const lParen = createToken({ name: 'LPAREN', pattern: /\(/, label: '(' });
const rParen = createToken({ name: 'RPAREN', pattern: /\)/, label: ')' });

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

const tcpPort = createToken({ name: 'TCP_PORT', pattern: /tcp\.port/, label: 'tcp.port' });
const udpPort = createToken({ name: 'UDP_PORT', pattern: /udp\.port/, label: 'udp.port' });

const payload = createToken({ name: 'PAYLOAD', pattern: /payload/, label: 'payload' });
const endReason = createToken({ name: 'END_REASON', pattern: /end\.reason/, label: 'end.reason' });
const handler = createToken({ name: 'HANDLER', pattern: /handler/, label: 'handler' });
const transport = createToken({ name: 'TRANSPORT', pattern: /transport/, label: 'transport' });
const scanner = createToken({ name: 'SCANNER', pattern: /scanner/, label: 'scanner' });
const string = createToken({
    name: 'STRING',
    pattern: /\"(?:[^"\\\r\n]|\\.)*\"/,
    label: '"msg"',
});

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

    lParen,
    rParen,

    contains,
    matches,
    matchesSmb,

    ipv4,
    port,

    ipSrc,
    tcpPort,
    udpPort,
    endReason,
    handler,
    transport,
    scanner,

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
            {
                ALT: () => {
                    this.CONSUME(lParen);
                    this.SUBRULE(this.query);
                    this.CONSUME(rParen);
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
        this.CONSUME(payload);
        this.SUBRULE(this.searchOperator);
        this.CONSUME(string);
    });

    private searchOperator = this.RULE('searchOperator', () => {
        this.OR([
            { ALT: () => this.CONSUME(contains) },
            { ALT: () => this.CONSUME(matches) },
            { ALT: () => this.CONSUME(matchesSmb) },
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
        // ip.dst was removed: dstHost is stripped from every event, so it could never match.
        this.CONSUME(ipSrc);
    });

    private stringItemClause = this.RULE('stringItemClause', () => {
        this.OR([
            { ALT: () => this.CONSUME(endReason) },
            { ALT: () => this.CONSUME(handler) },
            { ALT: () => this.CONSUME(transport) },
            { ALT: () => this.CONSUME(scanner) },
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
            {
                ALT: () => {
                    this.SUBRULE(this.stringItemClause);
                    this.SUBRULE2(this.binaryOperator);
                    this.CONSUME(string);
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
    // Patterns in `payload matches` clauses that the browser could not compile. The grammar
    // cannot catch these, and an invalid pattern must not reach filterEvent.
    regexErrors: string[];
}

export const productions: Record<string, Rule> = parser.getGAstProductions();

// create the HTML Text
export const serializedGrammar = parser.getSerializedGastProductions();

const QUERY_START_SUGGESTIONS = [
    'tcp.port',
    'udp.port',
    'ip.src',
    'end.reason',
    'handler',
    'transport',
    'scanner',
    'payload',
    'not',
    '(',
];

const FIELD_SUGGESTIONS = [
    'tcp.port',
    'udp.port',
    'ip.src',
    'end.reason',
    'handler',
    'transport',
    'scanner',
    'payload',
    '(',
];

const OPERATOR_SUGGESTIONS = ['eq', 'ne', '==', '!='];
const BOOLEAN_SUFFIX_SUGGESTIONS = ['and', 'or'];

function tokenMatches(token: IToken, ...types: TokenType[]): boolean {
    return types.includes(token.tokenType);
}

// Number of groups opened but not yet closed, so ')' is only offered when it can match.
function openGroups(tokens: IToken[]): number {
    let depth = 0;
    for (const token of tokens) {
        if (tokenMatches(token, lParen)) {
            depth++;
        } else if (tokenMatches(token, rParen) && depth > 0) {
            depth--;
        }
    }
    return depth;
}

function computeSuggestions(assistTokens: IToken[]): string[] {
    if (assistTokens.length === 0) {
        return QUERY_START_SUGGESTIONS;
    }

    const lastToken = assistTokens[assistTokens.length - 1];

    if (tokenMatches(lastToken, and, or, lParen)) {
        return QUERY_START_SUGGESTIONS;
    }

    if (tokenMatches(lastToken, not)) {
        return FIELD_SUGGESTIONS;
    }

    if (tokenMatches(lastToken, tcpPort, udpPort, ipSrc, endReason, handler, transport, scanner)) {
        return OPERATOR_SUGGESTIONS;
    }

    if (tokenMatches(lastToken, eq, ne, eqSmb, neSmb)) {
        const fieldToken = assistTokens[assistTokens.length - 2];
        if (fieldToken && tokenMatches(fieldToken, tcpPort, udpPort)) {
            return ['<PORT>'];
        }
        if (fieldToken && tokenMatches(fieldToken, ipSrc)) {
            return ['<IP>'];
        }
        if (fieldToken && tokenMatches(fieldToken, endReason)) {
            return ['"timeout"'];
        }
        if (fieldToken && tokenMatches(fieldToken, handler)) {
            return ['"http"'];
        }
        if (fieldToken && tokenMatches(fieldToken, transport)) {
            return ['"tcp"'];
        }
        if (fieldToken && tokenMatches(fieldToken, scanner)) {
            return ['"censys"'];
        }
        return [];
    }

    if (tokenMatches(lastToken, port, ipv4, string, rParen)) {
        return openGroups(assistTokens) > 0
            ? [...BOOLEAN_SUFFIX_SUGGESTIONS, ')']
            : BOOLEAN_SUFFIX_SUGGESTIONS;
    }

    if (tokenMatches(lastToken, payload)) {
        return ['contains', 'matches'];
    }

    if (tokenMatches(lastToken, contains)) {
        return ['"msg"'];
    }

    if (tokenMatches(lastToken, matches, matchesSmb)) {
        return ['"regex"'];
    }

    return [];
}

// parseStringLiteral turns a STRING token's image into the text it stands for. It lives here,
// next to the token it decodes, so both the parser and the filter can use it.
export function parseStringLiteral(image: string): string {
    const raw = image.slice(1, -1);
    try {
        return JSON.parse(image);
    } catch {
        return raw.replace(/\\(.)/g, '$1');
    }
}

// A `matches` operand is a pattern, not text, so it keeps its backslashes. Unescaping it the
// way parseStringLiteral does would quietly change what it matches: "\.php" would become
// ".php", where . matches any character, "\d+" would become "d+", and "\b" would become a
// backspace instead of a word boundary. The pattern is therefore the literal text between the
// quotes; \\" needs no unescaping either, since a regular expression reads it as a quote.
const inlineFlags = /^\(\?([ims]+)\)/;

export function compilePattern(image: string): RegExp {
    const source = image.slice(1, -1);
    // A leading (?i), the way Go and Python write it. JavaScript takes flags separately, so
    // the group is translated rather than passed through, which RegExp would reject.
    const inline = inlineFlags.exec(source);
    return inline ? new RegExp(source.slice(inline[0].length), inline[1]) : new RegExp(source);
}

// Each pattern is compiled once, while the parse result is still being built, and is then
// looked up by the token it came from. filterEvent never compiles anything itself, so it
// cannot disagree with what parseDSL accepted. A token lives exactly as long as the parsed
// filter that holds it, so the entry goes away with it.
const compiledPatterns = new WeakMap<IToken, RegExp>();

export function patternFor(token: IToken): RegExp {
    const pattern = compiledPatterns.get(token);
    if (!pattern) {
        // Only reachable if a filter is evaluated without parseDSL having accepted it.
        throw new Error('Pattern for search clause was never compiled');
    }
    return pattern;
}

// compilePatterns walks the searchClause nodes of a parsed filter and compiles the pattern of
// every `matches` / `~` clause, so one that does not compile is reported like a syntax error
// rather than thrown while events are being filtered. It walks the same nodes filterEvent
// evaluates; re-scanning the token stream instead would let the two drift apart.
function compilePatterns(node: CstNode, errors: string[]): void {
    if (node.name === 'searchClause') {
        const children = node.children as SearchClauseCstChildren;
        const operator = children.searchOperator[0].children;
        if (!operator.MATCHES && !operator.MATCHES_SMB) {
            return;
        }
        const token = children.STRING[0];
        try {
            compiledPatterns.set(token, compilePattern(token.image));
        } catch (e) {
            errors.push(e instanceof Error ? e.message : String(e));
        }
        return;
    }
    for (const children of Object.values(node.children)) {
        for (const child of children) {
            if ('children' in child) {
                compilePatterns(child, errors);
            }
        }
    }
}

export function parseDSL(
    text: string,
    filterState: FilterState = { suggestions: [], partialToken: null },
): ParseResult {
    const lexResult = queryLexer.tokenize(text);

    if (lexResult.errors.length > 0) {
        filterState.suggestions = [];
        filterState.partialToken = null;
        return {
            lexErrors: lexResult.errors,
            parseErrors: [],
            regexErrors: [],
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

    // Only worth walking once the clauses are known to be well formed; a failed parse leaves
    // a partial tree whose searchClause nodes may be missing the operator or the pattern.
    const regexErrors: string[] = [];
    if (parser.errors.length === 0) {
        compilePatterns(cst, regexErrors);
    }

    return {
        // This is a pure grammar, the value will be undefined until we add embedded actions
        // or enable automatic CST creation.
        cst: cst as QueryCstNode,
        lexErrors: lexResult.errors,
        parseErrors: parser.errors,
        regexErrors,
    };
}
