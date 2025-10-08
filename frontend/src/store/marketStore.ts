import { create } from 'zustand'
import api, { APIResponse, toastEmitter } from '@/utils/api'; // 引入 APIResponse 类型 和 toastEmitter

// 服务类型定义
export interface ServiceType {
    id: string;
    name: string;
    version: string;
    description: string;
    source: 'npm' | 'pypi' | 'local' | 'recommended';
    downloads?: number;
    stars?: number;
    author?: string | { name?: string; email?: string; url?: string };
    repositoryUrl?: string;
    homepage?: string;
    lastUpdated?: string;
    isInstalled?: boolean;
    installed_service_id?: number;
    envVars: EnvVarType[];
    readme?: string;
    // 添加缺少的字段
    display_name?: string;
    health_status?: string;
    health_details?: string;
    last_health_check?: string;
    enabled?: boolean;
    // 添加 RPD 限制和用户请求统计相关字段
    rpd_limit?: number;
    user_daily_request_count?: number;
    remaining_requests?: number;
    // 添加编辑功能需要的字段
    command?: string;
    headers_json?: string;
    type?: 'stdio' | 'sse' | 'streamableHttp';
    args_json?: string;
    default_envs_json?: string;
}

// 详细服务类型定义
export interface ServiceDetailType extends ServiceType {
    isLoading: boolean;
    mcpConfig?: {
        mcpServers?: {
            [key: string]: {
                env?: { [key: string]: string };
            }
        }
    };
}

// 环境变量类型
export interface EnvVarType {
    name: string;
    value?: string;
    description?: string;
    isSecret?: boolean;
    isRequired?: boolean;
    optional?: boolean;
    defaultValue?: string;
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

// 新：市场来源类型
export type MarketSource = 'npm' | 'pypi';

// 定义 Store 状态类型
interface MarketState {
    // 搜索相关
    searchTerm: string;
    searchResults: ServiceType[];
    isSearching: boolean;
    activeMarketTab: MarketSource; // 新增：当前激活的市场选项卡

    // 已安装服务
    installedServices: ServiceType[];

    // 服务详情
    selectedService: ServiceDetailType | null;
    isLoadingDetails: boolean;

    // 安装任务
    installTasks: Record<string, InstallTask>;
    installPollTimers: Record<string, ReturnType<typeof setTimeout>>;
    // 卸载任务状态
    uninstallTasks: Record<string, UninstallTask>;

    // 操作方法
    setSearchTerm: (term: string) => void;
    setActiveMarketTab: (tab: MarketSource) => void; // 新增：设置激活的市场选项卡
    searchServices: (sourceArg?: 'installed') => Promise<void>; // 修改 searchServices 签名
    fetchInstalledServices: () => Promise<void>;
    selectService: (serviceId: string) => void;
    fetchServiceDetails: (serviceId: string, packageName?: string, packageManager?: string) => Promise<void>;
    clearSelectedService: () => void;
    updateEnvVar: (serviceId: string, envVarName: string, value: string) => void;
    installService: (serviceId: string, envVars: { [key: string]: string }) => Promise<any>;
    updateInstallProgress: (serviceId: string, log: string) => void;
    updateInstallStatus: (serviceId: string, status: InstallStatus, error?: string) => void;
    pollInstallationStatus: (serviceId: string, taskId: string) => void;
    cancelInstallTask: (serviceId: string) => void;
    uninstallService: (serviceId: number) => Promise<void>;
    toggleService: (serviceId: string) => Promise<void>;
    checkServiceHealth: (serviceId: string) => Promise<void>;
}

// 创建 Store
export const useMarketStore = create<MarketState>((set, get) => ({
    // 初始状态
    searchTerm: '',
    searchResults: [],
    isSearching: false,
    activeMarketTab: 'npm', // 新增：初始化 activeMarketTab
    installedServices: [],
    selectedService: null,
    isLoadingDetails: false,
    installTasks: {},
    installPollTimers: {},
    uninstallTasks: {},

    // 操作方法
    setSearchTerm: (term) => set({ searchTerm: term }),

    setActiveMarketTab: (tab) => set({ activeMarketTab: tab }), // 新增：实现 setActiveMarketTab

    searchServices: async (sourceArg) => { // Renamed param from 'source' to 'sourceArg' to avoid conflict if any local var named 'source'
        const { searchTerm, fetchInstalledServices, activeMarketTab } = get();
        set({ isSearching: true });

        // 如果是请求已安装的服务
        if (sourceArg === 'installed') {
            await fetchInstalledServices();
            set({ isSearching: false });
            return;
        }

        // 如果搜索词为空 (且不是请求已安装服务)，则清空结果并停止
        if (!searchTerm) {
            set({ searchResults: [], isSearching: false });
            return;
        }

        try {
            // sources 由 activeMarketTab 决定
            const currentSearchSource = activeMarketTab;
            const response = await api.get(`/mcp_market/search?query=${encodeURIComponent(searchTerm)}&sources=${currentSearchSource}`) as APIResponse<any>;

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

                        // 创建服务ID
                        const serviceId = item.name + '-' + item.package_manager;

                        // Check if the service is installed (this local check might be redundant if backend provides definitive is_installed and installed_service_id)
                        // const isInstalled = installedServices.some(installed =>
                        //     installed.id === serviceId ||
                        //     (installed.name === item.name && installed.source === item.package_manager)
                        // );

                        return {
                            id: serviceId,
                            name: item.name || 'Unknown Name',
                            version: item.version || '0.0.0',
                            description: item.description || '',
                            source: item.package_manager || 'unknown',
                            downloads: item.downloads, // Use item.downloads for weekly downloads
                            stars: typeof item.github_stars === 'number' ? item.github_stars : undefined,
                            author: author,
                            repositoryUrl: item.repository_url,
                            homepage: item.homepage,
                            lastUpdated: item.last_publish, // Assuming this is 'last_updated' from backend
                            isInstalled: item.is_installed || false, // Rely on backend's is_installed
                            installed_service_id: item.installed_service_id, // Map new field from backend
                            envVars: item.env_vars ? item.env_vars.map((envDef: any) => ({
                                name: envDef.name,
                                description: envDef.description,
                                isSecret: envDef.is_secret,
                                isRequired: !envDef.optional,
                                defaultValue: envDef.default_value,
                                value: item.mcp_config?.mcpServers?.[envDef.name]?.env?.[envDef.name] !== undefined ? item.mcp_config.mcpServers[envDef.name].env[envDef.name] : (envDef.default_value || '')
                            })) : [],
                            readme: item.readme || '',
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
                    installed_service_id: info.installed_service_id,
                }));

                set({
                    installedServices,
                    searchResults: get().searchResults
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

    fetchServiceDetails: async (_serviceId, packageName, packageManager) => {
        set({ isLoadingDetails: true });

        try {
            if (!packageName || !packageManager) {
                throw new Error('Package name or manager not provided');
            }

            const response = await api.get(`/mcp_market/package_details?package_name=${encodeURIComponent(packageName)}&package_manager=${packageManager}`) as APIResponse<any>;

            if (response.success && response.data) {
                const details = response.data; // `details` here is the *entire* data object from API

                // Safely extract savedValues from mcp_config
                let savedValues: Record<string, string> = {};
                if (details.mcp_config && details.mcp_config.mcpServers) {
                    const serverKeys = Object.keys(details.mcp_config.mcpServers);
                    if (serverKeys.length > 0) {
                        // Assuming the first server key is the relevant one
                        const primaryServerKey = serverKeys[0];
                        savedValues = details.mcp_config.mcpServers[primaryServerKey]?.env || {};
                    }
                }

                // 将环境变量转换为前端格式 (unused but kept for consistency)
                // const _envVars = details.env_vars ? details.env_vars.map((envDef: any) => ({
                //     name: envDef.name,
                //     description: envDef.description,
                //     isSecret: envDef.is_secret,
                //     isRequired: !envDef.optional, // Mapping 'optional' to 'isRequired'
                //     defaultValue: envDef.default_value,
                //     value: savedValues[envDef.name] !== undefined ? savedValues[envDef.name] : (envDef.default_value || '') // Populate value
                // })) : [];

                set({
                    selectedService: {
                        id: `${details.details.name}-${packageManager}`,
                        name: details.details.name,
                        version: details.details.version || 'N/A',
                        description: details.details.description || '',
                        source: packageManager as 'npm' | 'pypi',
                        downloads: details.downloads,
                        stars: details.stars,
                        author: details.author,
                        repositoryUrl: details.repository_url,
                        homepage: details.details.homepage,
                        lastUpdated: details.last_publish,
                        isInstalled: details.is_installed || false,
                        installed_service_id: details.installed_service_id,
                        envVars: details.env_vars ? details.env_vars.map((envDef: any) => ({
                            name: envDef.name,
                            description: envDef.description,
                            isSecret: envDef.is_secret,
                            isRequired: !envDef.optional,
                            optional: envDef.optional,
                            defaultValue: envDef.default_value,
                            value: details.is_installed && savedValues[envDef.name] !== undefined ? savedValues[envDef.name] : ''
                        })) : [],
                        readme: details.readme || '',
                        mcpConfig: details.mcp_config,
                        isLoading: false,
                    },
                    isLoadingDetails: false,
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
        get().cancelInstallTask(serviceId);

        const { searchResults, installedServices } = get();

        // 直接从 searchResults 或 installedServices 查找 service 信息
        const service = [...searchResults, ...installedServices].find(s => s.id === serviceId);
        if (!service) {
            console.error(`Service with ID ${serviceId} not found for installation.`);
            toastEmitter.emit({
                variant: "destructive",
                title: "Error",
                description: `Service with ID ${serviceId} not found.`
            });
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
            if (response.success === false && response.data && Array.isArray(response.data.required_env_vars) && response.data.required_env_vars.length > 0) {
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

    cancelInstallTask: (serviceId) => {
        const existingTimer = get().installPollTimers[serviceId];
        if (existingTimer) {
            clearTimeout(existingTimer);
        }
        set(state => {
            const nextTimers = { ...state.installPollTimers };
            delete nextTimers[serviceId];

            const nextTasks = { ...state.installTasks };
            const task = nextTasks[serviceId];
            if (task) {
                delete nextTasks[serviceId];
            }

            return {
                installPollTimers: nextTimers,
                installTasks: nextTasks,
            };
        });
    },

    pollInstallationStatus: async (serviceId, taskId) => {
        const clearExistingTimer = () => {
            const existingTimer = get().installPollTimers[serviceId];
            if (existingTimer) {
                clearTimeout(existingTimer);
                set(state => {
                    const nextTimers = { ...state.installPollTimers };
                    delete nextTimers[serviceId];
                    return { installPollTimers: nextTimers };
                });
            }
        };

        const scheduleNextPoll = () => {
            clearExistingTimer();
            const timerId = setTimeout(() => {
                get().pollInstallationStatus(serviceId, taskId);
            }, 5000);
            set(state => ({
                installPollTimers: {
                    ...state.installPollTimers,
                    [serviceId]: timerId,
                },
            }));
        };

        const currentTask = get().installTasks[serviceId];
        if (!currentTask || currentTask.status !== 'installing') {
            clearExistingTimer();
            return;
        }

        const { searchResults, installedServices } = get();
        const service = [...searchResults, ...installedServices].find(s => s.id === serviceId);
        const serviceDisplayName = service?.name || serviceId;

        try {
            const response = await api.get(`/mcp_market/install_status/${taskId}`) as APIResponse<any>;
            if (response.success && response.data) {
                const { status, logs = [], error_message } = response.data;
                logs.forEach((log: string) => get().updateInstallProgress(serviceId, log));

                if (status === 'completed') {
                    clearExistingTimer();
                    get().updateInstallStatus(serviceId, 'success');
                    toastEmitter.emit({
                        title: "Installation Complete",
                        description: `${serviceDisplayName} has been successfully installed.`
                    });
                    get().fetchInstalledServices();
                } else if (status === 'failed') {
                    clearExistingTimer();
                    get().updateInstallStatus(serviceId, 'error', error_message || 'Installation failed');
                    toastEmitter.emit({
                        variant: "destructive",
                        title: "Installation Failed",
                        description: error_message || `Failed to install ${serviceDisplayName}.`
                    });
                } else if (status === 'pending' || status === 'running' || status === 'installing') {
                    scheduleNextPoll();
                } else {
                    scheduleNextPoll();
                }
            } else {
                clearExistingTimer();
                console.warn("Failed to poll installation status, or no data:", response.message);
                get().updateInstallStatus(serviceId, 'error', 'Polling failed. Check server logs.');
                toastEmitter.emit({
                    variant: "destructive",
                    title: "Polling Error",
                    description: `Unable to get installation status for ${serviceDisplayName}.`
                });
            }
        } catch (error) {
            clearExistingTimer();
            console.error('Polling error:', error);
            get().updateInstallStatus(serviceId, 'error', 'Failed to poll installation status.');
            toastEmitter.emit({
                variant: "destructive",
                title: "Polling Error",
                description: `Error polling installation status for ${serviceDisplayName}.`
            });
        }
    },

    uninstallService: async (serviceId: number): Promise<void> => {
        const { searchResults, installedServices } = get();
        const serviceIdString = String(serviceId);
        const allServices = [...searchResults, ...installedServices];
        const service = allServices.find(s => {
            if (s.installed_service_id === serviceId) return true;
            return String(s.installed_service_id) === serviceIdString || s.id === serviceIdString;
        });
        const serviceDisplayName = service?.name || String(serviceId);

        set(state => ({
            uninstallTasks: {
                ...state.uninstallTasks,
                [serviceIdString]: { status: 'uninstalling', error: undefined },
            },
        }));

        try {
            const response = await api.post('/mcp_market/uninstall', {
                service_id: serviceId,
            }) as APIResponse<any>;

            if (response.success) {
                set(state => {
                    const updatedSearchResults = state.searchResults.map(s =>
                        (s.installed_service_id === serviceId || s.id === serviceIdString) ? { ...s, isInstalled: false, installed_service_id: undefined } : s
                    );

                    const updatedInstalledServices = state.installedServices.filter(s =>
                        !(s.installed_service_id === serviceId || s.id === serviceIdString)
                    );

                    // 修复：使用service.id来删除installTask，而不是serviceIdString
                    const restInstallTasks = { ...state.installTasks };
                    if (service && service.id && restInstallTasks[service.id]) {
                        delete restInstallTasks[service.id];
                    }
                    // 也尝试删除serviceIdString的任务（以防万一）
                    if (restInstallTasks[serviceIdString]) {
                        delete restInstallTasks[serviceIdString];
                    }

                    return {
                        uninstallTasks: {
                            ...state.uninstallTasks,
                            [serviceIdString]: { status: 'idle', error: undefined },
                        },
                        installTasks: restInstallTasks,
                        searchResults: updatedSearchResults,
                        installedServices: updatedInstalledServices,
                    };
                });
                toastEmitter.emit({
                    title: "Uninstall Complete",
                    description: `Service ${serviceDisplayName} has been uninstalled.`
                });
            } else {
                const errorMsg = response.message || `Failed to uninstall ${serviceDisplayName}.`;
                set(state => ({
                    uninstallTasks: {
                        ...state.uninstallTasks,
                        [serviceIdString]: { status: 'error', error: errorMsg },
                    },
                }));
                toastEmitter.emit({
                    variant: "destructive",
                    title: "Uninstall Failed",
                    description: errorMsg
                });
            }
        } catch (error: any) {
            const errorMsg = error.response?.data?.message || error.message || `An unknown error occurred while uninstalling ${serviceDisplayName}.`;
            set(state => ({
                uninstallTasks: {
                    ...state.uninstallTasks,
                    [serviceIdString]: { status: 'error', error: errorMsg },
                },
            }));
            toastEmitter.emit({
                variant: "destructive",
                title: "Uninstall Error",
                description: errorMsg
            });
            throw error;
        }
    },

    toggleService: async (serviceId: string): Promise<void> => {
        const numericServiceId = parseInt(serviceId, 10);
        if (isNaN(numericServiceId)) {
            toastEmitter.emit({
                variant: "destructive",
                title: "Toggle Failed",
                description: "Invalid Service ID format."
            });
            return;
        }

        // Get the current state of the service *before* toggling
        const currentServices = get().installedServices;
        const serviceToToggle = currentServices.find(s => s.id === serviceId);

        if (!serviceToToggle) {
            toastEmitter.emit({
                variant: "destructive",
                title: "Toggle Failed",
                description: "Service not found in the local store."
            });
            return; // Or throw new Error("Service not found");
        }
        const wasEnabled = serviceToToggle.enabled;

        try {
            const response = await api.post(`/mcp_services/${numericServiceId}/toggle`) as APIResponse<any>;

            if (response.success) {
                toastEmitter.emit({
                    title: "Service Status Updated",
                    description: response.message || "Service status has been successfully changed."
                });

                // Update the service in the store directly
                const newEnabledState = !wasEnabled;
                set(state => ({
                    installedServices: state.installedServices.map(service =>
                        service.id === serviceId
                            ? { ...service, enabled: newEnabledState }
                            : service
                    ),
                }));

                // If the service was just enabled, trigger an immediate health check
                if (newEnabledState) { // This means wasEnabled was false, and now it's true
                    try {
                        await get().checkServiceHealth(serviceId);
                        // Toast for health check success/failure is handled within checkServiceHealth
                    } catch (healthCheckError: any) {
                        // Optional: additional error handling specific to health check after toggle
                        console.error(`Immediate health check for service ${serviceId} failed after enabling:`, healthCheckError);
                        // No need to re-throw or show another toast if checkServiceHealth already does.
                    }
                }
                // No longer fetching all services: await get().fetchInstalledServices();
            } else {
                throw new Error(response.message || 'Failed to toggle service status');
            }
        } catch (error: any) {
            const errorMsg = error.response?.data?.message || error.message || 'An unknown error occurred while toggling service status.';
            toastEmitter.emit({
                variant: "destructive",
                title: "Toggle Service Failed",
                description: errorMsg
            });
            throw error; // Re-throw to allow component-level error handling if needed
        }
    },

    checkServiceHealth: async (serviceId: string): Promise<void> => {
        const numericServiceId = parseInt(serviceId, 10);
        if (isNaN(numericServiceId)) {
            toastEmitter.emit({
                variant: "destructive",
                title: "Health Check Failed",
                description: "Invalid Service ID format."
            });
            return;
        }

        try {
            const response = await api.post(`/mcp_services/${numericServiceId}/health/check`) as APIResponse<any>;

            if (response.success && response.data) {
                toastEmitter.emit({
                    title: "Health Check Complete",
                    description: "Service health status has been updated."
                });

                // 直接使用返回值更新对应服务的健康状态，而不是重新拉取整个列表
                const { health_status, health_details, last_checked } = response.data;

                set(state => ({
                    installedServices: state.installedServices.map(service =>
                        service.id === serviceId || service.id === numericServiceId.toString()
                            ? {
                                ...service,
                                health_status,
                                health_details: typeof health_details === 'object' ? JSON.stringify(health_details) : health_details,
                                last_health_check: last_checked
                            }
                            : service
                    )
                }));
            } else {
                throw new Error(response.message || 'Failed to check service health');
            }
        } catch (error: any) {
            const errorMsg = error.response?.data?.message || error.message || 'An unknown error occurred while checking service health.';
            toastEmitter.emit({
                variant: "destructive",
                title: "Health Check Failed",
                description: errorMsg
            });
            throw error;
        }
    },
})); 
