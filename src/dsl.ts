import { createToken, Lexer, CstParser, Rule } from "../node_modules/chevrotain/lib/chevrotain"

// DSL representing a simplified version of Wiresharks Display Filter


// comparing
const eq = createToken({ name: "EQUAL", pattern: /eq/, label: 'eq' })

// logical
const and = createToken({ name: "AND", pattern: /and/, label: 'and' })

// literals
const dot = createToken({ name: "DOT", pattern: /\./, label: '.' })
const integer = createToken({ name: 'INTEGER', pattern: /-?(?:0|[1-9]\d*)/ });


// identifiers
const tcp = createToken({ name: "TCP", pattern: /tcp/, label: 'tcp' })
const udp = createToken({ name: "UDP", pattern: /udp/, label: 'udp' })
const port = createToken({ name: "PORT", pattern: /port/, label: 'port' })

const whiteSpace = createToken({
    name: "WhiteSpace",
    pattern: /\s+/,
    group: Lexer.SKIPPED
})

let allTokens = [
    whiteSpace,
    eq,
    and,
    dot,
    integer,
    // identifiers after keywords
    tcp,
    udp,
    port,
]

let dfLexer = new Lexer(allTokens)

//let inputText = "tcp.port eq 23 and tcp.port eq 445"

class dfParserTypeScript extends CstParser {
    constructor() {
        super(allTokens)
        this.performSelfAnalysis()
    }

    public df = this.RULE("portFilter", () => {
        this.SUBRULE(this.protocol)
        this.CONSUME(dot)
        this.CONSUME(port)
        this.CONSUME(eq)
        this.CONSUME(integer)
    })

    private protocol = this.RULE("protocol", () => {
        this.OR([
            // using ES6 Arrow functions to reduce verbosity.
            { ALT: () => this.CONSUME(tcp) },
            { ALT: () => this.CONSUME(udp) }
        ])
    })
}

const parser = new dfParserTypeScript()

export const productions: Record<string, Rule> = parser.getGAstProductions()

export function parseDSL(text: string) {
    const lexResult = dfLexer.tokenize(text)
    // setting a new input will RESET the parser instance's state.
    parser.input = lexResult.tokens
    // any top level rule may be used as an entry point
    const cst = parser.df()

    return {
        // This is a pure grammar, the value will be undefined until we add embedded actions
        // or enable automatic CST creation.
        cst: cst,
        lexErrors: lexResult.errors,
        parseErrors: parser.errors
    }
}