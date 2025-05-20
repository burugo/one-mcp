import { create } from 'zustand'
import { ReactNode } from 'react';
import api, { APIResponse, toastEmitter } from '@/utils/api'; // 引入 APIResponse 类型 和 toastEmitter

// 服务类型定义
export interface ServiceType {
    id: string;
    name: string;
    description: string;
    version: string;
    source: string;  // package_manager (npm, pypi, etc.)
    author: string;
    stars?: number; // Preferably GitHub stars
    npmScore?: number; // For npm-specific score, if no GitHub stars
    homepageUrl?: string; // URL to GitHub page or official website
    icon?: ReactNode;
    isInstalled?: boolean;
    health_status?: string;
    display_name?: string;
    enabled?: boolean;
}

// 详细服务类型定义
export interface ServiceDetailType extends ServiceType {
    readme: string;
    envVars: EnvVarType[];
}

// 环境变量类型
export interface EnvVarType {
    name: string;
    description: string;
    isSecret: boolean;
    isRequired: boolean;
    defaultValue?: string;
    value?: string;
}

// 安装状态类型
export type InstallStatus = 'idle' | 'installing' | 'success' | 'error';

// 安装任务类型
export interface InstallTask {
    serviceId: string;
    status: InstallStatus;
    logs: string[];
    error?: string;
    taskId?: string;
}

// 卸载任务状态
export interface UninstallTask {
    status: 'idle' | 'uninstalling' | 'error';
    error?: string;
}

// 定义 Store 状态类型
interface MarketState {
    // 搜索相关
    searchTerm: string;
    searchResults: ServiceType[];
    isSearching: boolean;
    activeTab: 'all' | 'npm' | 'pypi' | 'recommended' | 'installed';

    // 已安装服务
    installedServices: ServiceType[];

    // 服务详情
    selectedService: ServiceDetailType | null;
    isLoadingDetails: boolean;

    // 安装任务
    installTasks: Record<string, InstallTask>;
    // 卸载任务状态
    uninstallTasks: Record<string, UninstallTask>;

    // 操作方法
    setSearchTerm: (term: string) => void;
    setActiveTab: (tab: 'all' | 'npm' | 'pypi' | 'recommended' | 'installed') => void;
    searchServices: () => Promise<void>;
    fetchInstalledServices: () => Promise<void>;

    selectService: (serviceId: string) => void;
    fetchServiceDetails: (serviceId: string, packageName?: string, packageManager?: string) => Promise<void>;
    clearSelectedService: () => void;

    updateEnvVar: (serviceId: string, envVarName: string, value: string) => void;

    installService: (serviceId: string, envVars: { [key: string]: string }) => Promise<any>;
    updateInstallProgress: (serviceId: string, log: string) => void;
    updateInstallStatus: (serviceId: string, status: InstallStatus, error?: string) => void;
    pollInstallationStatus: (serviceId: string, taskId: string) => void;

    uninstallService: (serviceId: string) => Promise<void>;
}

// 创建 Store
export const useMarketStore = create<MarketState>((set, get) => ({
    // 初始状态
    searchTerm: '',
    searchResults: [],
    isSearching: false,
    activeTab: 'all',
    installedServices: [],
    selectedService: null,
    isLoadingDetails: false,
    installTasks: {},
    uninstallTasks: {},

    // 操作方法
    setSearchTerm: (term) => set({ searchTerm: term }),

    setActiveTab: (tab) => set({ activeTab: tab }),

    searchServices: async () => {
        const { searchTerm, activeTab } = get();
        set({ isSearching: true });

        // If searchTerm is empty (original logic)
        // and not looking at installed tab, clear results and stop.
        if (!searchTerm && activeTab !== 'installed') { // Reverted to original searchTerm check
            set({ searchResults: [], isSearching: false });
            return;
        }

        try {
            // 构建搜索来源参数
            let sources = '';
            switch (activeTab) {
                case 'all': sources = 'npm,pypi,recommended'; break;
                case 'installed': return get().fetchInstalledServices();
                default: sources = activeTab;
            }

            const response = await api.get(`/mcp_market/search?query=${encodeURIComponent(searchTerm)}&sources=${sources}`) as APIResponse<any>;

            if (response.success) {
                if (Array.isArray(response.data)) {
                    // Map backend data to frontend ServiceType
                    const mappedResults: ServiceType[] = response.data.map((item: any) => {
                        let author = item.author || 'Unknown Author';
                        const homepageUrl = item.homepage;

                        if ((!item.author || item.author.toLowerCase() === 'unknown') && homepageUrl && homepageUrl.includes('github.com')) {
                            try {
                                const url = new URL(homepageUrl);
                                const pathParts = url.pathname.split('/').filter(part => part.length > 0);
                                if (pathParts.length > 0) {
                                    author = pathParts[0]; // Usually the owner/org
                                }
                            } catch (e) {
                                console.warn('Failed to parse homepage URL for author:', homepageUrl, e);
                            }
                        }

                        return {
                            id: item.name + '-' + item.package_manager, // Create a unique ID
                            name: item.name || 'Unknown Name',
                            description: item.description || '',
                            version: item.version || '0.0.0',
                            source: item.package_manager || 'unknown',
                            author: author,
                            stars: typeof item.github_stars === 'number' ? item.github_stars : undefined,
                            npmScore: typeof item.score === 'number' ? item.score : undefined,
                            homepageUrl: homepageUrl,
                            isInstalled: item.is_installed || false,
                            health_status: item.HealthStatus || item.health_status || '',
                            display_name: item.DisplayName || item.display_name || item.Name || item.name,
                            enabled: typeof item.Enabled === 'boolean' ? item.Enabled : undefined,
                        };
                    });
                    set({ searchResults: mappedResults });
                } else {
                    set({ searchResults: [] });
                }
            } else {
                throw new Error(response.message || 'Failed to search services');
            }
        } catch (error) {
            console.error('Search error:', error);
            // 可以设置一个错误状态，但这里暂时不处理
        } finally {
            set({ isSearching: false });
        }
    },

    fetchInstalledServices: async () => {
        set({ isSearching: true });

        try {
            const response = await api.get('/mcp_market/installed') as APIResponse<any>;

            if (response.success && Array.isArray(response.data)) {
                // 直接用后端返回的数组，保留所有字段
                const installedServices = response.data.map((info: any) => ({
                    ...info,
                    id: info.id || info.Name || info.name, // 兼容各种 id 字段
                    name: info.Name || info.name,
                    display_name: info.DisplayName || info.display_name || info.Name || info.name,
                    description: info.Description || info.description || '',
                    version: info.InstalledVersion || info.version || 'unknown',
                    source: info.PackageManager || info.package_manager || 'unknown',
                    author: 'Installed',
                    stars: 0,
                    npmScore: undefined,
                    homepageUrl: undefined,
                    isInstalled: true,
                    env_vars: info.env_vars || {},
                    health_status: info.HealthStatus || info.health_status || '',
                    health_details: info.HealthDetails || info.health_details || '',
                    enabled: typeof info.Enabled === 'boolean' ? info.Enabled : undefined,
                }));

                set({
                    installedServices,
                    searchResults: get().activeTab === 'installed' ? installedServices : get().searchResults
                });
            } else {
                throw new Error(response.message || 'Failed to fetch installed services');
            }
        } catch (error) {
            console.error('Fetch installed services error:', error);
        } finally {
            set({ isSearching: false });
        }
    },

    selectService: (serviceId) => {
        // 这里仅设置选择的服务ID，具体的加载逻辑在 fetchServiceDetails 中
        const service = [...get().searchResults, ...get().installedServices].find(s => s.id === serviceId);

        if (service) {
            get().fetchServiceDetails(serviceId, service.name, service.source);
        }
    },

    fetchServiceDetails: async (serviceId, packageName, packageManager) => {
        set({ isLoadingDetails: true });

        try {
            if (!packageName || !packageManager) {
                throw new Error('Package name or manager not provided');
            }

            const response = await api.get(`/mcp_market/package_details?package_name=${encodeURIComponent(packageName)}&package_manager=${packageManager}`) as APIResponse<any>;

            if (response.success && response.data) {
                const details = response.data;

                // 将环境变量转换为前端格式
                const envVars = details.env_vars.map((env: any) => ({
                    name: env.name,
                    description: env.description,
                    isSecret: env.is_secret,
                    isRequired: !env.optional,
                    defaultValue: env.default_value
                }));

                set({
                    selectedService: {
                        id: serviceId,
                        name: details.details.name,
                        description: details.details.description,
                        version: details.details.version,
                        source: packageManager,
                        author: details.details.author || 'Unknown',
                        stars: details.details.stars || 0,
                        npmScore: details.details.npm_score || undefined,
                        homepageUrl: details.details.homepage_url || undefined,
                        readme: details.readme || '',
                        envVars
                    }
                });
            } else {
                throw new Error(response.message || 'Failed to fetch service details');
            }
        } catch (error) {
            console.error('Fetch service details error:', error);
        } finally {
            set({ isLoadingDetails: false });
        }
    },

    clearSelectedService: () => set({ selectedService: null }),

    updateEnvVar: (serviceId, envVarName, value) => {
        const { selectedService } = get();

        if (selectedService && selectedService.id === serviceId) {
            const updatedEnvVars = selectedService.envVars.map(env =>
                env.name === envVarName ? { ...env, value } : env
            );

            set({ selectedService: { ...selectedService, envVars: updatedEnvVars } });
        }
    },

    installService: async (serviceId, envVars): Promise<any> => {
        const { searchResults, installedServices, activeTab } = get();

        // 直接从 searchResults 或 installedServices 查找 service 信息
        const service = [...searchResults, ...installedServices].find(s => s.id === serviceId);
        if (!service) {
            console.error(`Service with ID ${serviceId} not found for installation.`);
            toastEmitter.emit({ variant: "destructive", title: "错误", description: `未能找到ID为 ${serviceId} 的服务。` });
            return;
        }

        // Determine source_type based on activeTab or other logic if needed
        const sourceType = 'marketplace';

        const currentTaskState = get().installTasks[serviceId];
        if (currentTaskState && currentTaskState.status === 'installing') {
            console.warn(`Installation for ${service.name} (${serviceId}) is already in progress.`);
            return; // Prevent re-initiating installation
        }

        // Initialize or reset task state
        set(state => ({
            installTasks: {
                ...state.installTasks,
                [serviceId]: {
                    serviceId, // Store serviceId which is the unique ID
                    status: 'installing',
                    logs: ['Installation initiated...'],
                    error: undefined,
                    taskId: undefined,
                }
            }
        }));

        try {
            const requestBody = {
                source_type: sourceType,
                package_name: service.name,
                package_manager: service.source,
                version: service.version,
                user_provided_env_vars: envVars,
                display_name: service.name,
                service_description: service.description,
            };
            const response = await api.post('/mcp_market/install_or_add_service', requestBody) as APIResponse<any>;
            // RESTful: 如果需要补充 env vars，直接返回 response（完整 APIResponse）
            if (response.success === true && response.data && Array.isArray(response.data.required_env_vars) && response.data.required_env_vars.length > 0) {
                return response;
            }

            if (!response.success || !response.data) {
                throw new Error(response.message || 'Installation setup failed');
            }

            const { mcp_service_id, task_id, status } = response.data;
            const effectiveTaskId = task_id || mcp_service_id;

            if (status === 'already_installed_instance_added') {
                get().updateInstallStatus(serviceId, 'success', 'Service instance added successfully.');
                get().fetchInstalledServices();
                get().clearSelectedService();
            } else if (effectiveTaskId) {
                set(state => ({
                    installTasks: {
                        ...state.installTasks,
                        [serviceId]: {
                            ...state.installTasks[serviceId],
                            taskId: effectiveTaskId,
                            logs: [...(state.installTasks[serviceId]?.logs || []), `Installation task submitted (Task ID: ${effectiveTaskId}). Polling for status...`]
                        }
                    }
                }));
                get().pollInstallationStatus(serviceId, effectiveTaskId);
            } else {
                // 如果没有 task_id 或 mcp_service_id，且不是 required_env_vars，说明后端有问题
                throw new Error('No task_id or mcp_service_id received from backend to start polling.');
            }

        } catch (error: any) {
            const errorMessage = error?.response?.message || error.message || '';
            console.error('Install service error:', error);
            get().updateInstallStatus(serviceId, 'error', errorMessage || 'An unknown error occurred during installation setup.');
            throw error;
        }
    },

    updateInstallProgress: (serviceId, log) => {
        const task = get().installTasks[serviceId];

        if (task) {
            set({
                installTasks: {
                    ...get().installTasks,
                    [serviceId]: {
                        ...task,
                        logs: [...task.logs, log]
                    }
                }
            });
        }
    },

    updateInstallStatus: (serviceId, status, error) => {
        set(state => ({
            installTasks: {
                ...state.installTasks,
                [serviceId]: {
                    ...state.installTasks[serviceId],
                    status: status,
                    error: error,
                }
            }
        }));
    },

    pollInstallationStatus: async (serviceId, taskId) => {
        const { searchResults, installedServices } = get();
        const service = [...searchResults, ...installedServices].find(s => s.id === serviceId);
        const serviceDisplayName = service?.name || serviceId;

        try {
            const response = await api.get(`/mcp_market/install_status/${taskId}`) as APIResponse<any>;
            if (response.success && response.data) {
                const { status, logs = [], error_message } = response.data;
                logs.forEach((log: string) => get().updateInstallProgress(serviceId, log));
                if (status === 'completed') {
                    get().updateInstallStatus(serviceId, 'success');
                    toastEmitter.emit({ title: "安装完成", description: `${serviceDisplayName} 已成功安装。` });
                    get().fetchInstalledServices();
                } else if (status === 'failed') {
                    get().updateInstallStatus(serviceId, 'error', error_message || 'Installation failed');
                    toastEmitter.emit({ variant: "destructive", title: "安装失败", description: error_message || `未能安装 ${serviceDisplayName}。` });
                } else if (status === 'pending' || status === 'running' || status === 'installing') {
                    // 继续轮询，不弹 toast
                    setTimeout(() => get().pollInstallationStatus(serviceId, taskId), 5000);
                } else {
                    // 其他未知状态，继续轮询
                    setTimeout(() => get().pollInstallationStatus(serviceId, taskId), 5000);
                }
            } else {
                // 只有在 response.success 为 false 或 data 缺失时才弹 toast
                console.warn("Failed to poll installation status, or no data:", response.message);
                get().updateInstallStatus(serviceId, 'error', 'Polling failed. Check server logs.');
                toastEmitter.emit({ variant: "destructive", title: "轮询错误", description: `无法获取 ${serviceDisplayName} 的安装状态。` });
            }
        } catch (error) {
            console.error('Polling error:', error);
            get().updateInstallStatus(serviceId, 'error', 'Failed to poll installation status.');
            toastEmitter.emit({ variant: "destructive", title: "轮询错误", description: `轮询 ${serviceDisplayName} 安装状态时出错。` });
        }
    },

    uninstallService: async (serviceId: string) => {
        const { searchResults, installedServices } = get();
        const service = [...searchResults, ...installedServices].find(s => s.id === serviceId);
        const serviceDisplayName = service?.name || serviceId;
        const packageName = service?.name || serviceId;
        const packageManager = service?.source || '';

        set(state => ({
            uninstallTasks: {
                ...state.uninstallTasks,
                [serviceId]: { status: 'uninstalling', error: undefined },
            },
        }));
        toastEmitter.emit({ title: "正在卸载", description: `服务 ${serviceDisplayName} 正在卸载中...` });

        try {
            // 修正为 POST /api/mcp_market/uninstall，body 传 package_name, package_manager
            const response = await api.post('/mcp_market/uninstall', {
                package_name: packageName,
                package_manager: packageManager,
            }) as APIResponse<any>;
            if (response.success) {
                set(state => {
                    const updatedSearchResults = state.searchResults.map(s =>
                        s.id === serviceId ? { ...s, isInstalled: false } : s
                    );
                    // 清理 installTasks[serviceId]
                    const { [serviceId]: _, ...restInstallTasks } = state.installTasks;
                    return {
                        uninstallTasks: {
                            ...state.uninstallTasks,
                            [serviceId]: { status: 'idle', error: undefined },
                        },
                        installTasks: restInstallTasks,
                        searchResults: updatedSearchResults,
                    };
                });
                toastEmitter.emit({ title: "卸载成功", description: `服务 ${serviceDisplayName} 已卸载。` });
                // get().searchServices(); // 可选：如需强制刷新
            } else {
                const errorMsg = response.message || `Failed to uninstall ${serviceDisplayName}.`;
                set(state => ({
                    uninstallTasks: {
                        ...state.uninstallTasks,
                        [serviceId]: { status: 'error', error: errorMsg },
                    },
                }));
                toastEmitter.emit({ variant: "destructive", title: "卸载失败", description: errorMsg });
            }
        } catch (error: any) {
            const errorMsg = error.response?.data?.message || error.message || `An unknown error occurred while uninstalling ${serviceDisplayName}.`;
            set(state => ({
                uninstallTasks: {
                    ...state.uninstallTasks,
                    [serviceId]: { status: 'error', error: errorMsg },
                },
            }));
            toastEmitter.emit({ variant: "destructive", title: "卸载错误", description: errorMsg });
        }
    },
})); 