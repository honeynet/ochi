module.exports = {
    transform: {
        '^.+\\.svelte$': [
            'svelte-jester',
            {
                preprocess: true,
            },
        ],
        '^.+\\.js$': 'babel-jest',
        '^.+\\.ts$': 'ts-jest',
    },
    moduleFileExtensions: ['js', 'ts', 'svelte'],
    moduleNameMapper: {
        '^@chevrotain/utils$': '<rootDir>/node_modules/@chevrotain/utils/lib/src/api.js',
        '^@chevrotain/gast$': '<rootDir>/node_modules/@chevrotain/gast/lib/src/api.js',
        '^@chevrotain/regexp-to-ast$':
            '<rootDir>/node_modules/@chevrotain/regexp-to-ast/lib/src/api.js',
        '^@chevrotain/cst-dts-gen$':
            '<rootDir>/node_modules/@chevrotain/cst-dts-gen/lib/src/api.js',
    },
    setupFiles: ['<rootDir>/jest.setup.js'],
    setupFilesAfterEnv: ['@testing-library/jest-dom/extend-expect'],
    testEnvironment: 'jsdom',
    globals: {
        'ts-jest': {
            tsconfig: 'tsconfig.jest.json',
        },
    },
    transformIgnorePatterns: [
        'node_modules/(?!(@chevrotain|chevrotain|@ngrx|(?!deck.gl)|ng-dynamic))',
    ],
};
