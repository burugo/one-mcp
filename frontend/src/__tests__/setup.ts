import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, beforeAll, vi } from 'vitest'

// 每个测试后清理
afterEach(() => {
    cleanup()
})

// Mock全局对象
beforeAll(() => {
    // Mock window.matchMedia
    Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: vi.fn().mockImplementation(query => ({
            matches: false,
            media: query,
            onchange: null,
            addListener: vi.fn(),
            removeListener: vi.fn(),
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
            dispatchEvent: vi.fn(),
        })),
    })

    // Mock IntersectionObserver
    global.IntersectionObserver = vi.fn().mockImplementation(() => ({
        observe: vi.fn(),
        unobserve: vi.fn(),
        disconnect: vi.fn(),
    }))

    // Mock ResizeObserver
    global.ResizeObserver = vi.fn().mockImplementation(() => ({
        observe: vi.fn(),
        unobserve: vi.fn(),
        disconnect: vi.fn(),
    }))

    // Mock localStorage
    const localStorageMock = {
        getItem: vi.fn(),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn(),
    }
    Object.defineProperty(window, 'localStorage', {
        value: localStorageMock
    })

    // Mock sessionStorage
    const sessionStorageMock = {
        getItem: vi.fn(),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn(),
    }
    Object.defineProperty(window, 'sessionStorage', {
        value: sessionStorageMock
    })

    // Mock global API calls
    vi.mock('@/utils/api', async (importOriginal) => {
        const actual = await importOriginal() as any
        return {
            ...actual,
            default: {
                ...actual?.default,
                get: vi.fn((url: string) => {
                    if (url === '/option/') {
                        return Promise.resolve({
                            success: true,
                            data: [{ key: 'ServerAddress', value: 'http://localhost:3003' }]
                        })
                    }
                    // For other GET requests, you might return a default mock or throw an error
                    return Promise.resolve({ success: true, data: {} })
                }),
                // Mock other methods like post, patch, delete if needed globally
            },
        }
    })
}) 