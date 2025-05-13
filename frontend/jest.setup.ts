import '@testing-library/jest-dom';

// Polyfill for TextEncoder/TextDecoder for Jest environment
import { TextEncoder, TextDecoder } from 'util';

if (typeof global.TextEncoder === 'undefined') {
    global.TextEncoder = TextEncoder;
}
if (typeof global.TextDecoder === 'undefined') {
    (global as any).TextDecoder = TextDecoder;
}

// Mock window.matchMedia for Jest environment (used by ThemeToggle)
Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: jest.fn().mockImplementation(query => ({
        matches: false, // Default to light mode or non-matching query
        media: query,
        onchange: null,
        addListener: jest.fn(), // Deprecated but still used in some libraries
        removeListener: jest.fn(), // Deprecated
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        dispatchEvent: jest.fn(),
    })),
}); 