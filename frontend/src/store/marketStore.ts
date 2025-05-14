import { create } from 'zustand'
import { ReactNode } from 'react';

// 服务类型定义
export interface ServiceType {
    id: string;
    name: string;
    description: string;
    version: string;
    source: string;
    author: string;
    downloads: number;
    stars: number;
    icon?: ReactNode;
    isInstalled?: boolean;
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

    // 操作方法
    setSearchTerm: (term: string) => void;
    setActiveTab: (tab: 'all' | 'npm' | 'pypi' | 'recommended' | 'installed') => void;
    searchServices: () => Promise<void>;
    fetchInstalledServices: () => Promise<void>;

    selectService: (serviceId: string) => void;
    fetchServiceDetails: (serviceId: string, packageName?: string, packageManager?: string) => Promise<void>;
    clearSelectedService: () => void;

    updateEnvVar: (serviceId: string, envVarName: string, value: string) => void;

    installService: (serviceId: string, envVars: { [key: string]: string }) => Promise<void>;
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

    // 操作方法
    setSearchTerm: (term) => set({ searchTerm: term }),

    setActiveTab: (tab) => set({ activeTab: tab }),

    searchServices: async () => {
        const { searchTerm, activeTab } = get();
        set({ isSearching: true });

        try {
            // 构建搜索来源参数
            let sources = '';
            switch (activeTab) {
                case 'all': sources = 'npm,pypi,recommended'; break;
                case 'installed': return get().fetchInstalledServices();
                default: sources = activeTab;
            }

            // 调用搜索 API
            const response = await fetch(`/api/mcp_market/search?query=${encodeURIComponent(searchTerm)}&sources=${sources}`);

            if (!response.ok) {
                throw new Error('Search failed');
            }

            const data = await response.json();
            if (data.code === 0 && data.data) {
                set({ searchResults: data.data });
            } else {
                throw new Error(data.msg || 'Failed to search services');
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
            const response = await fetch('/api/mcp_market/installed');

            if (!response.ok) {
                throw new Error('Failed to fetch installed services');
            }

            const data = await response.json();
            if (data.code === 0 && data.data) {
                // 将已安装服务数据转换为 ServiceType 格式
                const installedServices = Object.entries(data.data).map(([packageName, info]: [string, any]) => ({
                    id: info.id || packageName,
                    name: packageName,
                    description: info.description || `MCP Server: ${packageName}`,
                    version: info.version || 'unknown',
                    source: info.package_manager || 'unknown',
                    author: 'Installed',
                    downloads: 0,
                    stars: 0,
                    isInstalled: true
                }));

                set({
                    installedServices,
                    searchResults: get().activeTab === 'installed' ? installedServices : get().searchResults
                });
            } else {
                throw new Error(data.msg || 'Failed to fetch installed services');
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

            const response = await fetch(`/api/mcp_market/package_details?package_name=${encodeURIComponent(packageName)}&package_manager=${packageManager}`);

            if (!response.ok) {
                throw new Error('Failed to fetch service details');
            }

            const data = await response.json();
            if (data.code === 0 && data.data) {
                const details = data.data;

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
                        downloads: details.details.downloads || 0,
                        stars: details.details.stars || 0,
                        readme: details.readme || '',
                        envVars
                    }
                });
            } else {
                throw new Error(data.msg || 'Failed to fetch service details');
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

    installService: async (serviceId, envVars) => {
        const { selectedService } = get();

        if (!selectedService) return;

        // 初始化安装任务状态
        set({
            installTasks: {
                ...get().installTasks,
                [serviceId]: {
                    serviceId,
                    status: 'installing',
                    logs: ['Starting installation...']
                }
            }
        });

        try {
            // 发送安装请求
            const response = await fetch('/api/mcp_market/install_or_add_service', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    source_type: 'marketplace',
                    package_name: selectedService.name,
                    package_manager: selectedService.source,
                    env_vars: envVars
                }),
            });

            const data = await response.json();
            if (!response.ok || data.code !== 0) {
                throw new Error(data.msg || 'Installation failed');
            }

            // 获取安装任务 ID 和服务 ID
            const { task_id } = data.data;

            // 更新任务 ID
            set({
                installTasks: {
                    ...get().installTasks,
                    [serviceId]: {
                        ...get().installTasks[serviceId],
                        taskId: task_id,
                        logs: [...get().installTasks[serviceId].logs, 'Installation started on server...']
                    }
                }
            });

            // 开始轮询安装状态
            get().pollInstallationStatus(serviceId, task_id);

        } catch (error) {
            console.error('Installation error:', error);
            get().updateInstallStatus(
                serviceId,
                'error',
                error instanceof Error ? error.message : String(error)
            );
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
        const task = get().installTasks[serviceId];

        if (task) {
            set({
                installTasks: {
                    ...get().installTasks,
                    [serviceId]: {
                        ...task,
                        status,
                        error,
                        logs: error
                            ? [...task.logs, `Error: ${error}`]
                            : (status === 'success'
                                ? [...task.logs, 'Installation completed successfully!']
                                : task.logs)
                    }
                }
            });

            // 如果安装成功，刷新已安装服务列表
            if (status === 'success') {
                get().fetchInstalledServices();
            }
        }
    },

    pollInstallationStatus: (serviceId, taskId) => {
        const interval = setInterval(async () => {
            try {
                // 检查任务状态是否已不再是"installing"
                const currentTask = get().installTasks[serviceId];
                if (!currentTask || currentTask.status !== 'installing') {
                    clearInterval(interval);
                    return;
                }

                const response = await fetch(`/api/mcp_market/installation_status?task_id=${taskId}`);
                const data = await response.json();

                if (data.code === 0 && data.data) {
                    // 更新安装日志
                    if (data.data.output) {
                        get().updateInstallProgress(serviceId, data.data.output);
                    }

                    // 检查状态
                    if (data.data.status === 'completed') {
                        clearInterval(interval);
                        get().updateInstallStatus(serviceId, 'success');
                    } else if (data.data.status === 'failed') {
                        clearInterval(interval);
                        get().updateInstallStatus(serviceId, 'error', data.data.error);
                    }
                }
            } catch (error) {
                console.error('Status polling error:', error);
                // 不要在这里设置错误状态，继续轮询
            }
        }, 1000); // 每秒轮询一次

        // 设置超时，防止无限轮询
        setTimeout(() => {
            clearInterval(interval);

            // 检查任务是否仍在安装中
            const currentTask = get().installTasks[serviceId];
            if (currentTask && currentTask.status === 'installing') {
                get().updateInstallStatus(
                    serviceId,
                    'error',
                    'Installation timed out. Please check service status.'
                );
            }
        }, 120000); // 2分钟超时
    },

    uninstallService: async (serviceId) => {
        const service = get().installedServices.find(s => s.id === serviceId);

        if (!service) return;

        try {
            const response = await fetch('/api/mcp_market/uninstall_service', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    package_name: service.name,
                    package_manager: service.source
                }),
            });

            const data = await response.json();
            if (!response.ok || data.code !== 0) {
                throw new Error(data.msg || 'Uninstallation failed');
            }

            // 刷新已安装服务列表
            get().fetchInstalledServices();

        } catch (error) {
            console.error('Uninstallation error:', error);
            // 可以设置一个错误状态或显示通知
        }
    }
})); 