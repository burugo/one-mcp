import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useMarketStore } from './marketStore'
import api from '@/utils/api'

// Mock API
vi.mock('@/utils/api', () => ({
    default: {
        get: vi.fn(),
        post: vi.fn(),
    },
    toastEmitter: {
        emit: vi.fn(),
    },
}))

describe('MarketStore', () => {
    beforeEach(() => {
        // Reset store state before each test
        useMarketStore.setState({
            searchTerm: '',
            searchResults: [],
            isSearching: false,
            activeMarketTab: 'npm',
            installedServices: [],
            selectedService: null,
            isLoadingDetails: false,
            installTasks: {},
            uninstallTasks: {},
        })
        vi.clearAllMocks()
    })

    afterEach(() => {
        vi.clearAllMocks()
    })

    describe('fetchServiceDetails', () => {
        it('should handle null env_vars without throwing error', async () => {
            // Mock API response with null env_vars (the bug scenario)
            const mockResponse = {
                success: true,
                data: {
                    details: {
                        name: 'test-package',
                        version: '1.0.0',
                        description: 'Test package',
                        homepage: 'https://example.com',
                    },
                    env_vars: null, // This was causing the bug
                    downloads_last_month: 1000,
                    stars: 50,
                    author: 'Test Author',
                    repository_url: 'https://github.com/test/repo',
                    last_publish: '2024-01-01',
                    score: 0.8,
                    is_installed: false,
                    installed_service_id: null,
                    readme: 'Test readme',
                    mcp_config: null,
                }
            }

            vi.mocked(api.get).mockResolvedValue(mockResponse)

            const store = useMarketStore.getState()

            // This should not throw an error
            await expect(
                store.fetchServiceDetails('test-id', 'test-package', 'npm')
            ).resolves.not.toThrow()

            // Verify the service was set correctly with empty envVars
            const selectedService = useMarketStore.getState().selectedService
            expect(selectedService).toBeDefined()
            expect(selectedService?.envVars).toEqual([])
            expect(selectedService?.name).toBe('test-package')
        })

        it('should handle undefined env_vars without throwing error', async () => {
            // Mock API response with undefined env_vars
            const mockResponse = {
                success: true,
                data: {
                    details: {
                        name: 'test-package',
                        version: '1.0.0',
                        description: 'Test package',
                        homepage: 'https://example.com',
                    },
                    // env_vars is undefined (not included in response)
                    downloads_last_month: 1000,
                    stars: 50,
                    author: 'Test Author',
                    repository_url: 'https://github.com/test/repo',
                    last_publish: '2024-01-01',
                    score: 0.8,
                    is_installed: false,
                    installed_service_id: null,
                    readme: 'Test readme',
                    mcp_config: null,
                }
            }

            vi.mocked(api.get).mockResolvedValue(mockResponse)

            const store = useMarketStore.getState()

            await expect(
                store.fetchServiceDetails('test-id', 'test-package', 'npm')
            ).resolves.not.toThrow()

            const selectedService = useMarketStore.getState().selectedService
            expect(selectedService?.envVars).toEqual([])
        })

        it('should properly process valid env_vars array', async () => {
            // Mock API response with valid env_vars
            const mockResponse = {
                success: true,
                data: {
                    details: {
                        name: 'test-package',
                        version: '1.0.0',
                        description: 'Test package',
                        homepage: 'https://example.com',
                    },
                    env_vars: [
                        {
                            name: 'API_KEY',
                            description: 'API Key for service',
                            is_secret: true,
                            optional: false,
                            default_value: '',
                        },
                        {
                            name: 'DEBUG_MODE',
                            description: 'Enable debug mode',
                            is_secret: false,
                            optional: true,
                            default_value: 'false',
                        }
                    ],
                    downloads_last_month: 1000,
                    stars: 50,
                    author: 'Test Author',
                    repository_url: 'https://github.com/test/repo',
                    last_publish: '2024-01-01',
                    score: 0.8,
                    is_installed: false,
                    installed_service_id: null,
                    readme: 'Test readme',
                    mcp_config: null,
                }
            }

            vi.mocked(api.get).mockResolvedValue(mockResponse)

            const store = useMarketStore.getState()
            await store.fetchServiceDetails('test-id', 'test-package', 'npm')

            const selectedService = useMarketStore.getState().selectedService
            expect(selectedService?.envVars).toHaveLength(2)
            expect(selectedService?.envVars[0]).toMatchObject({
                name: 'API_KEY',
                description: 'API Key for service',
                isSecret: true,
                isRequired: true, // !optional
                defaultValue: '',
                value: '',
            })
            expect(selectedService?.envVars[1]).toMatchObject({
                name: 'DEBUG_MODE',
                description: 'Enable debug mode',
                isSecret: false,
                isRequired: false, // !optional
                defaultValue: 'false',
                value: 'false',
            })
        })

        it('should handle API error gracefully', async () => {
            const mockError = new Error('API Error')
            vi.mocked(api.get).mockRejectedValue(mockError)

            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => { })

            const store = useMarketStore.getState()
            await store.fetchServiceDetails('test-id', 'test-package', 'npm')

            expect(consoleSpy).toHaveBeenCalledWith('Fetch service details error:', mockError)
            expect(useMarketStore.getState().isLoadingDetails).toBe(false)

            consoleSpy.mockRestore()
        })

        it('should handle missing package name or manager', async () => {
            const store = useMarketStore.getState()

            // Test missing package name
            await store.fetchServiceDetails('test-id', '', 'npm')
            expect(useMarketStore.getState().isLoadingDetails).toBe(false)

            // Test missing package manager
            await store.fetchServiceDetails('test-id', 'test-package', '')
            expect(useMarketStore.getState().isLoadingDetails).toBe(false)
        })

        it('should handle unsuccessful API response', async () => {
            const mockResponse = {
                success: false,
                message: 'Package not found',
                data: null,
            }

            vi.mocked(api.get).mockResolvedValue(mockResponse)
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => { })

            const store = useMarketStore.getState()
            await store.fetchServiceDetails('test-id', 'test-package', 'npm')

            expect(consoleSpy).toHaveBeenCalled()
            expect(useMarketStore.getState().isLoadingDetails).toBe(false)

            consoleSpy.mockRestore()
        })

        it('should extract saved values from mcp_config correctly', async () => {
            const mockResponse = {
                success: true,
                data: {
                    details: {
                        name: 'test-package',
                        version: '1.0.0',
                        description: 'Test package',
                        homepage: 'https://example.com',
                    },
                    env_vars: [
                        {
                            name: 'API_KEY',
                            description: 'API Key',
                            is_secret: true,
                            optional: false,
                            default_value: '',
                        }
                    ],
                    mcp_config: {
                        mcpServers: {
                            'test-server': {
                                env: {
                                    'API_KEY': 'saved-api-key-value'
                                }
                            }
                        }
                    },
                    downloads_last_month: 1000,
                    stars: 50,
                    author: 'Test Author',
                    repository_url: 'https://github.com/test/repo',
                    last_publish: '2024-01-01',
                    score: 0.8,
                    is_installed: true,
                    installed_service_id: 123,
                    readme: 'Test readme',
                }
            }

            vi.mocked(api.get).mockResolvedValue(mockResponse)

            const store = useMarketStore.getState()
            await store.fetchServiceDetails('test-id', 'test-package', 'npm')

            const selectedService = useMarketStore.getState().selectedService
            expect(selectedService?.envVars[0].value).toBe('saved-api-key-value')
            expect(selectedService?.isInstalled).toBe(true)
            expect(selectedService?.installed_service_id).toBe(123)
        })
    })

    describe('basic store operations', () => {
        it('should set search term', () => {
            const store = useMarketStore.getState()
            store.setSearchTerm('test search')

            expect(useMarketStore.getState().searchTerm).toBe('test search')
        })

        it('should set active tab', () => {
            const store = useMarketStore.getState()
            store.setActiveMarketTab('npm')

            expect(useMarketStore.getState().activeMarketTab).toBe('npm')
        })

        it('should clear selected service', () => {
            // Set a selected service first
            useMarketStore.setState({
                selectedService: {
                    id: 'test',
                    name: 'test',
                    version: '1.0.0',
                    description: 'test',
                    source: 'npm',
                    envVars: [],
                    isLoading: false,
                }
            })

            const store = useMarketStore.getState()
            store.clearSelectedService()

            expect(useMarketStore.getState().selectedService).toBeNull()
        })

        it('should update env var value', () => {
            const mockService = {
                id: 'test-service',
                name: 'test',
                version: '1.0.0',
                description: 'test',
                source: 'npm' as const,
                envVars: [
                    {
                        name: 'API_KEY',
                        value: 'old-value',
                        description: 'API Key',
                        isSecret: true,
                        isRequired: true,
                        defaultValue: '',
                    }
                ],
                isLoading: false,
            }

            useMarketStore.setState({ selectedService: mockService })

            const store = useMarketStore.getState()
            store.updateEnvVar('test-service', 'API_KEY', 'new-value')

            const updatedService = useMarketStore.getState().selectedService
            expect(updatedService?.envVars[0].value).toBe('new-value')
        })
    })
}) 