module.exports = {
    preset: 'ts-jest',
    testEnvironment: 'jsdom',
    moduleNameMapper: {
        '\\.(css|less|scss|sass)$': 'identity-obj-proxy', // Mock CSS imports
        '^@/(.*)$': '<rootDir>/src/$1', // Handle path aliases like @/components
        // Add mappings for React to ensure a single instance
        '^react$': '<rootDir>/node_modules/react',
        '^react-dom$': '<rootDir>/node_modules/react-dom',
    },
    setupFilesAfterEnv: ['<rootDir>/jest.setup.ts'],
    transform: {
        '^.+\\.(ts|tsx)?$': ['ts-jest', { tsconfig: '<rootDir>/tsconfig.json' }],
    },
    // Automatically clear mock calls, instances, contexts and results before every test
    clearMocks: true,
    // The directory where Jest should output its coverage files
    coverageDirectory: "coverage",
    // Indicates which provider should be used to instrument code for coverage
    coverageProvider: "v8",
}; 