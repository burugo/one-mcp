import { useEffect, useState, useRef } from 'react';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableHead, TableHeader, TableRow, TableCell } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Search, PlusCircle, Trash2, Plus, RotateCcw, Grid, List, Upload, Pencil } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { useNavigate } from 'react-router-dom';
import { useMarketStore, ServiceType } from '@/store/marketStore';
import ServiceConfigModal from '@/components/market/ServiceConfigModal';
import CustomServiceModal, { CustomServiceData } from '@/components/market/CustomServiceModal';
import EditServiceModal, { EditServiceData } from '@/components/market/EditServiceModal';
import BatchImportModal from '@/components/market/BatchImportModal';
import api, { APIResponse } from '@/utils/api';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { useTranslation } from 'react-i18next';
import { useAuth } from '@/contexts/AuthContext';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';

export function ServicesPage() {
    const { t } = useTranslation();
    const { toast } = useToast();
    const navigate = useNavigate();
    const { currentUser } = useAuth();
    const { installedServices: globalInstalledServices, fetchInstalledServices, uninstallService, toggleService, checkServiceHealth } = useMarketStore();
    const [configModalOpen, setConfigModalOpen] = useState(false);
    const [customServiceModalOpen, setCustomServiceModalOpen] = useState(false);
    const [batchImportModalOpen, setBatchImportModalOpen] = useState(false);
    const [editModalOpen, setEditModalOpen] = useState(false);
    const [selectedService, setSelectedService] = useState<ServiceType | null>(null);
    const [serviceToEdit, setServiceToEdit] = useState<ServiceType | null>(null);
    const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
    const [pendingUninstallId, setPendingUninstallId] = useState<string | null>(null);
    const [togglingServices, setTogglingServices] = useState<Set<string>>(new Set());
    const [checkingHealthServices, setCheckingHealthServices] = useState<Set<string>>(new Set());
    const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
    const [autoFillEnv, setAutoFillEnv] = useState('');

    const hasFetched = useRef(false);

    // 检查用户是否是管理员(role >= 10)
    const isAdmin = currentUser?.role && currentUser.role >= 10;

    useEffect(() => {
        if (!hasFetched.current) {
            fetchInstalledServices();
            hasFetched.current = true;
        }
    }, [fetchInstalledServices]);

    const allServices = globalInstalledServices;
    const activeServices = globalInstalledServices.filter(s => s.enabled === true);
    const inactiveServices = globalInstalledServices.filter(s => s.enabled === false);



    const handleUninstallClick = (serviceId: string) => {
        setPendingUninstallId(serviceId);
        setUninstallDialogOpen(true);
    };

    const handleUninstallConfirm = async () => {
        if (!pendingUninstallId) return;

        const numericServiceId = parseInt(pendingUninstallId, 10);
        if (isNaN(numericServiceId)) {
            toast({
                title: 'Uninstall Failed',
                description: 'Invalid Service ID format.',
                variant: 'destructive'
            });
            setUninstallDialogOpen(false);
            setPendingUninstallId(null);
            return;
        }

        setUninstallDialogOpen(false);

        try {
            await uninstallService(numericServiceId);
            toast({
                title: 'Uninstall Complete',
                description: 'Service has been successfully uninstalled.'
            });
            fetchInstalledServices();
        } catch (e: any) {
            toast({
                title: 'Uninstall Failed',
                description: e?.message || 'Unknown error',
                variant: 'destructive'
            });
        } finally {
            setPendingUninstallId(null);
        }
    };

    const parseEnvironments = (envStr?: string): Record<string, string> => {
        if (!envStr) return {};
        return envStr.split(/\r?\n|\\n/).reduce((acc, rawLine) => {
            const line = rawLine.trim();
            if (!line || !line.includes('=')) {
                return acc;
            }
            const [key, ...valueParts] = line.split('=');
            const trimmedKey = key.trim();
            const value = valueParts.join('=').trim();
            if (trimmedKey && value) {
                acc[trimmedKey] = value;
            }
            return acc;
        }, {} as Record<string, string>);
    };

    const tokenizeCommand = (command?: string): string[] => {
        if (!command) return [];
        const tokens: string[] = [];
        let current = '';
        let quote: '"' | "'" | '`' | null = null;

        for (let i = 0; i < command.length; i++) {
            const char = command[i];

            if (quote) {
                if (char === quote) {
                    quote = null;
                } else if (char === '\\' && quote === '"' && i + 1 < command.length) {
                    // Preserve escaped characters inside double quotes
                    current += command[i + 1];
                    i += 1;
                } else {
                    current += char;
                }
                continue;
            }

            if (char === '"' || char === "'" || char === '`') {
                quote = char;
                continue;
            }

            if (/\s/.test(char)) {
                if (current) {
                    tokens.push(current);
                    current = '';
                }
                continue;
            }

            if (char === '\\' && i + 1 < command.length) {
                current += command[i + 1];
                i += 1;
                continue;
            }

            current += char;
        }

        if (current) {
            tokens.push(current);
        }

        return tokens;
    };

    const extractPackageInfo = (tokens: string[]): { packageManager: '' | 'npm' | 'uv'; packageName: string } => {
        if (tokens.length === 0) {
            return { packageManager: '', packageName: '' };
        }

        const [managerToken, ...args] = tokens;
        if (managerToken !== 'npx' && managerToken !== 'uvx') {
            return { packageManager: '', packageName: '' };
        }

        const booleanFlags =
            managerToken === 'npx'
                ? new Set(['-y', '--yes', '--no-install', '--prefer-offline', '--quiet'])
                : new Set(['--quiet', '--preview', '--system', '--strict']);

        let packageName = '';
        for (let i = 0; i < args.length; i++) {
            const arg = args[i];

            if (arg === '--') {
                if (i + 1 < args.length) {
                    packageName = args[i + 1];
                }
                break;
            }

            if (arg.startsWith('-')) {
                const normalized = arg.includes('=') ? arg.split('=')[0] : arg;
                if (booleanFlags.has(normalized) || arg.includes('=')) {
                    continue;
                }

                if (i + 1 < args.length) {
                    i += 1;
                }
                continue;
            }

            packageName = arg;
            break;
        }

        const packageManager = managerToken === 'npx' ? 'npm' : 'uv';
        return { packageManager, packageName };
    };

    const handleToggleService = async (serviceId: string) => {
        if (togglingServices.has(serviceId)) {
            return; // 防止重复点击
        }

        setTogglingServices(prev => new Set(prev).add(serviceId));

        try {
            await toggleService(serviceId);
        } catch (error: any) {
            console.error('Toggle service failed:', error);
        } finally {
            setTogglingServices(prev => {
                const newSet = new Set(prev);
                newSet.delete(serviceId);
                return newSet;
            });
        }
    };

    const handleCheckServiceHealth = async (serviceId: string) => {
        if (checkingHealthServices.has(serviceId)) {
            return; // 防止重复点击
        }

        setCheckingHealthServices(prev => new Set(prev).add(serviceId));

        try {
            await checkServiceHealth(serviceId);
        } catch (error: any) {
            console.error('Check service health failed:', error);
        } finally {
            setCheckingHealthServices(prev => {
                const newSet = new Set(prev);
                newSet.delete(serviceId);
                return newSet;
            });
        }
    };

    const pollInstallationStatus = async (serviceId: string, serviceName: string) => {
        const maxAttempts = 30; // 最多轮询30次（约5分钟）
        let attempts = 0;

        const poll = async () => {
            try {
                attempts++;
                const response = await api.get(`/mcp_market/install_status/${serviceId}`) as APIResponse<any>;

                if (response.success && response.data) {
                    const status = response.data.status;

                    if (status === 'completed') {
                        toast({
                            title: t('customServiceModal.messages.installationSuccess'),
                            description: t('customServiceModal.messages.installationSuccessDescription', { serviceName })
                        });
                        fetchInstalledServices(); // 刷新列表
                        return;
                    } else if (status === 'failed') {
                        toast({
                            title: t('customServiceModal.messages.installationFailed'),
                            description: t('customServiceModal.messages.installationFailedDescription', {
                                serviceName,
                                error: response.data.error || t('customServiceModal.messages.unknownError')
                            }),
                            variant: 'destructive'
                        });
                        return;
                    } else if (status === 'installing' || status === 'pending') {
                        // 继续轮询
                        if (attempts < maxAttempts) {
                            setTimeout(poll, 10000); // 10秒后再次检查
                        } else {
                            toast({
                                title: t('customServiceModal.messages.installationTimeout'),
                                description: t('customServiceModal.messages.installationTimeoutDescription', { serviceName }),
                                variant: 'destructive'
                            });
                        }
                        return;
                    }
                }

                // 默认情况下继续轮询
                if (attempts < maxAttempts) {
                    setTimeout(poll, 10000);
                } else {
                    toast({
                        title: t('customServiceModal.messages.statusCheckTimeout'),
                        description: t('customServiceModal.messages.statusCheckTimeoutDescription', { serviceName }),
                        variant: 'destructive'
                    });
                }
            } catch (error: any) {
                console.error('Poll installation status error:', error);
                if (attempts < maxAttempts) {
                    setTimeout(poll, 10000); // 出错时也继续尝试
                } else {
                    toast({
                        title: t('customServiceModal.messages.statusCheckFailed'),
                        description: t('customServiceModal.messages.statusCheckFailedDescription', { serviceName }),
                        variant: 'destructive'
                    });
                }
            }
        };

        // 开始轮询
        poll();
    };

    const handleCreateCustomService = async (serviceData: CustomServiceData) => {
        try {
            let res;
            if (serviceData.type === 'stdio') {
                const command = serviceData.command?.trim();

                const commandTokens = tokenizeCommand(command);
                const customArgs = commandTokens.length > 1 ? commandTokens.slice(1) : [];
                const { packageManager, packageName } = extractPackageInfo(commandTokens);

                if (!packageManager || !packageName) {
                    throw new Error(t('customServiceModal.messages.parseCommandFailed'));
                }

                const payload = {
                    source_type: 'custom',
                    package_name: packageName,
                    package_manager: packageManager,
                    display_name: serviceData.name,
                    user_provided_env_vars: parseEnvironments(serviceData.environments),
                    custom_args: customArgs.length > 0 ? customArgs : undefined, // Send custom args if available
                };
                res = await api.post('/mcp_market/install_or_add_service', payload) as APIResponse<any>;

            } else {
                // For 'sse' and 'streamableHttp'
                res = await api.post('/mcp_market/custom_service', serviceData) as APIResponse<any>;
            }

            if (res.success) {
                if (serviceData.type === 'stdio') {
                    // stdio 服务通过 install_or_add_service API，需要等待安装完成
                    toast({
                        title: t('customServiceModal.messages.installationSubmitted'),
                        description: t('customServiceModal.messages.installationSubmittedDescription', { serviceName: serviceData.name })
                    });

                    // 轮询安装状态
                    const serviceId = res.data?.mcp_service_id;
                    if (serviceId) {
                        pollInstallationStatus(serviceId, serviceData.name);
                    }
                    // 对于 stdio 服务，立即关闭模态框，因为轮询会处理后续状态
                    setCustomServiceModalOpen(false);
                } else {
                    // sse 和 streamableHttp 服务直接创建完成
                    toast({
                        title: t('customServiceModal.messages.createSuccess'),
                        description: t('customServiceModal.messages.createSuccessDescription', { serviceName: serviceData.name })
                    });
                    // 延迟刷新列表，等待服务注册完成
                    setTimeout(async () => {
                        await fetchInstalledServices();
                        // 确保列表刷新完成后再关闭模态框
                        setCustomServiceModalOpen(false);
                    }, 1000);
                }
                return res.data;
            } else {
                // Handle case where success is false but response is HTTP 200 (missing env vars)
                if (res.success === false && res.data?.required_env_vars && Array.isArray(res.data.required_env_vars) && res.data.required_env_vars.length > 0) {
                    // Auto-fill environment variable if missing
                    const missingEnvs = res.data.required_env_vars;
                    const envVarToSet = `${missingEnvs[0]}=`;
                    setAutoFillEnv(envVarToSet);

                    toast({
                        title: t('customServiceModal.messages.createFailed'),
                        description: res.message || '缺少必需环境变量',
                        variant: 'destructive'
                    });
                } else {
                    toast({
                        title: t('customServiceModal.messages.createFailed'),
                        description: res.message || '创建失败',
                        variant: 'destructive'
                    });
                }
                throw new Error(res.message || '创建失败');
            }
        } catch (error: any) {
            // Extract the actual error message from the API response
            let errorMessage = t('customServiceModal.messages.unknownError');
            if (error.response?.data?.message) {
                errorMessage = error.response.data.message;
            } else if (error.message) {
                errorMessage = error.message;
            }

            toast({
                title: t('customServiceModal.messages.createFailed'),
                description: errorMessage,
                variant: 'destructive'
            });

            // Auto-fill environment variable if missing
            if (error.response?.data?.data?.required_env_vars) {
                const missingEnvs = error.response.data.data.required_env_vars;
                if (missingEnvs.length > 0) {
                    const envVarToSet = `${missingEnvs[0]}=`;
                    setAutoFillEnv(envVarToSet);
                }
            }

            // Re-throw the error so CustomServiceModal can handle its loading state
            throw error;
        }
    };

    const handleUpdateService = async (serviceData: EditServiceData) => {
        try {
            // Base fields that can always be updated
            const updatePayload: any = {
                display_name: serviceData.display_name,
                description: serviceData.description,
            };

            // Add type-specific fields
            if (serviceData.type === 'sse' || serviceData.type === 'streamableHttp') {
                // For SSE/HTTP services, URL is stored in the command field
                if (serviceData.url) {
                    updatePayload.command = serviceData.url;
                }

                // For non-stdio services, also update headers
                if (serviceData.headers) {
                    updatePayload.headers_json = JSON.stringify(
                        serviceData.headers.split('\n')
                            .filter(line => line.trim())
                            .reduce((acc, line) => {
                                const [key, ...valueParts] = line.split('=');
                                if (key?.trim() && valueParts.length > 0) {
                                    acc[key.trim()] = valueParts.join('=').trim();
                                }
                                return acc;
                            }, {} as Record<string, string>)
                    );
                }
            } else if (serviceData.type === 'stdio') {
                // For stdio services, update environment variables
                if (serviceData.envVars !== undefined) {
                    updatePayload.default_envs_json = JSON.stringify(
                        serviceData.envVars.split('\n')
                            .filter(line => line.trim())
                            .reduce((acc, line) => {
                                const [key, ...valueParts] = line.split('=');
                                if (key?.trim() && valueParts.length > 0) {
                                    acc[key.trim()] = valueParts.join('=').trim();
                                }
                                return acc;
                            }, {} as Record<string, string>)
                    );
                }
            }

            const response = await api.put(`/mcp_services/${serviceData.id}`, updatePayload) as APIResponse<any>;

            if (response.success) {
                toast({
                    title: 'Update Successful',
                    description: 'Service has been successfully updated.'
                });

                // Refresh the services list
                await fetchInstalledServices();

                // Close the modal
                setEditModalOpen(false);
                setServiceToEdit(null);
            } else {
                throw new Error(response.message || 'Update failed');
            }
        } catch (error: any) {
            let errorMessage = 'Unknown error';
            if (error.response?.data?.message) {
                errorMessage = error.response.data.message;
            } else if (error.message) {
                errorMessage = error.message;
            }

            toast({
                title: 'Update Failed',
                description: errorMessage,
                variant: 'destructive'
            });

            // Re-throw the error so EditServiceModal can handle its loading state
            throw error;
        }
    };

    // 渲染列表视图
    const renderListView = (services: ServiceType[]) => {
        if (services.length === 0) {
            return (
                <div className="text-center py-8 text-muted-foreground">
                    <p>{t('services.noServicesFound')}</p>
                </div>
            );
        }

        return (
            <Table>
                <TableHeader>
                    <TableRow>
                        <TableHead>{t('services.status')}</TableHead>
                        <TableHead>{t('services.serviceName')}</TableHead>
                        <TableHead>{t('services.description')}</TableHead>
                        <TableHead>{t('services.version')}</TableHead>
                        <TableHead>{t('services.healthStatus')}</TableHead>
                        <TableHead>{t('services.enabledStatus')}</TableHead>
                        <TableHead>{t('services.operations')}</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {services.map(service => (
                        <TableRow key={service.id}>
                            <TableCell>
                                <div className="flex items-center space-x-1">
                                    <div className={`w-2 h-2 rounded-full ${service.health_status === "healthy" || service.health_status === "Healthy"
                                        ? "bg-green-500"
                                        : "bg-gray-400"
                                        }`}></div>
                                    <button
                                        className="p-0.5 rounded hover:bg-gray-100 text-gray-500 hover:text-gray-700 transition-colors"
                                        onClick={() => handleCheckServiceHealth(service.id)}
                                        disabled={checkingHealthServices.has(service.id)}
                                        title={t('services.refreshHealthStatus')}
                                    >
                                        <RotateCcw size={12} className={checkingHealthServices.has(service.id) ? "animate-spin" : ""} />
                                    </button>
                                </div>
                            </TableCell>
                            <TableCell>
                                <div className="font-medium">{service.display_name || service.name}</div>
                                <div className="text-sm text-muted-foreground">{service.name}</div>
                            </TableCell>
                            <TableCell>
                                <div className="max-w-xs truncate text-sm text-muted-foreground">
                                    {service.description}
                                </div>
                            </TableCell>
                            <TableCell>
                                <Badge variant="outline">{service.version || 'unknown'}</Badge>
                            </TableCell>
                            <TableCell>
                                <Badge variant={service.health_status === "healthy" || service.health_status === "Healthy" ? "default" : "secondary"}>
                                    {service.health_status || 'unknown'}
                                </Badge>
                            </TableCell>
                            <TableCell>
                                <Switch
                                    checked={service.enabled || false}
                                    onCheckedChange={() => handleToggleService(service.id)}
                                    disabled={togglingServices.has(service.id)}
                                />
                            </TableCell>
                            <TableCell>
                                <div className="flex items-center space-x-2">
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}
                                    >
                                        {t('services.configure')}
                                    </Button>
                                    {isAdmin && (
                                        <>
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                onClick={() => { setServiceToEdit(service); setEditModalOpen(true); }}
                                            >
                                                {t('services.edit')}
                                            </Button>
                                            <button
                                                className="p-1 rounded hover:bg-red-100 text-red-500"
                                                onClick={() => handleUninstallClick(service.id)}
                                                title={t('services.uninstallService')}
                                            >
                                                <Trash2 size={16} />
                                            </button>
                                        </>
                                    )}
                                </div>
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        );
    };

    // 渲染网格视图
    const renderGridView = (services: ServiceType[]) => {
        if (services.length === 0) {
            return (
                <div className="col-span-3 text-center py-8 text-muted-foreground">
                    <p>{t('services.noServicesFound')}</p>
                </div>
            );
        }

        return services.map(service => (
            <Card key={service.id} className="border-border shadow-sm hover:shadow transition-shadow duration-200 bg-card/30 flex flex-col">
                <CardHeader>
                    <div className="flex items-center justify-between">
                        <div className="flex items-center">
                            <div className="bg-primary/10 p-2 rounded-md mr-3">
                                <Search className="w-6 h-6 text-primary" />
                            </div>
                            <div className="flex items-center space-x-2">
                                <div className="flex items-center space-x-1">
                                    <div className={`w-2 h-2 rounded-full ${service.health_status === "healthy" || service.health_status === "Healthy"
                                        ? "bg-green-500"
                                        : "bg-gray-400"
                                        }`}></div>
                                    <button
                                        className="p-0.5 rounded hover:bg-gray-100 text-gray-500 hover:text-gray-700 transition-colors"
                                        onClick={() => handleCheckServiceHealth(service.id)}
                                        disabled={checkingHealthServices.has(service.id)}
                                        title={t('services.refreshHealthStatus')}
                                    >
                                        <RotateCcw size={12} className={checkingHealthServices.has(service.id) ? "animate-spin" : ""} />
                                    </button>
                                </div>
                                <CardTitle className="text-lg">{service.display_name || service.name}</CardTitle>
                            </div>
                        </div>
                        {isAdmin && (
                            <div className="flex items-center space-x-1 ml-2">
                                <button
                                    className="p-1 rounded hover:bg-blue-100 text-blue-500"
                                    onClick={() => { setServiceToEdit(service); setEditModalOpen(true); }}
                                    title={t('services.editService')}
                                >
                                    <Pencil size={16} />
                                </button>
                                <button
                                    className="p-1 rounded hover:bg-red-100 text-red-500"
                                    onClick={() => handleUninstallClick(service.id)}
                                    title={t('services.uninstallService')}
                                >
                                    <Trash2 size={18} />
                                </button>
                            </div>
                        )}
                    </div>
                </CardHeader>
                <CardContent className="flex-grow">
                    <p className="text-sm text-muted-foreground line-clamp-2 mb-3">{service.description}</p>
                    {/* RPD Limit and Usage Display */}
                    {service.rpd_limit !== undefined && service.rpd_limit !== null && service.rpd_limit > 0 && (
                        <div className="mt-2 pt-2 border-t border-border/50">
                            <div className="flex items-center justify-between text-xs">
                                <span className="text-muted-foreground">{t('services.dailyRequests')}:</span>
                                <span className="font-medium">
                                    {service.user_daily_request_count || 0} / {service.rpd_limit}
                                </span>
                            </div>
                            <div className="mt-1 w-full bg-gray-200 rounded-full h-1.5 dark:bg-gray-700">
                                <div
                                    className={`h-1.5 rounded-full transition-all duration-300 ${(service.user_daily_request_count || 0) >= service.rpd_limit
                                        ? 'bg-red-500'
                                        : (service.user_daily_request_count || 0) >= service.rpd_limit * 0.8
                                            ? 'bg-yellow-500'
                                            : 'bg-green-500'
                                        }`}
                                    style={{
                                        width: `${Math.min((service.user_daily_request_count || 0) / service.rpd_limit * 100, 100)}%`
                                    }}
                                ></div>
                            </div>
                            {service.remaining_requests !== undefined && service.remaining_requests >= 0 && (
                                <div className="mt-1 text-xs text-muted-foreground">
                                    {service.remaining_requests} {t('services.requestsRemaining')}
                                </div>
                            )}
                        </div>
                    )}
                </CardContent>
                <CardFooter className="flex justify-between items-end mt-auto">
                    <Button variant="outline" size="sm" className="h-6" onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}>{t('services.configure')}</Button>
                    <Switch
                        checked={service.enabled || false}
                        onCheckedChange={() => handleToggleService(service.id)}
                        disabled={togglingServices.has(service.id)}
                    />
                </CardFooter>
            </Card>
        ));
    };

    return (
        <div className="w-full space-y-8">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h2 className="text-3xl font-bold tracking-tight">{t('services.mcpServices')}</h2>
                    <p className="text-muted-foreground mt-1">{t('services.manageAndConfigure')}</p>
                </div>
                <div className="flex items-center space-x-2">
                    {/* 视图切换按钮 */}
                    <div className="flex items-center border rounded-md">
                        <Button
                            variant={viewMode === 'grid' ? 'default' : 'ghost'}
                            size="sm"
                            onClick={() => setViewMode('grid')}
                            className="rounded-r-none"
                        >
                            <Grid className="w-4 h-4" />
                        </Button>
                        <Button
                            variant={viewMode === 'list' ? 'default' : 'ghost'}
                            size="sm"
                            onClick={() => setViewMode('list')}
                            className="rounded-l-none"
                        >
                            <List className="w-4 h-4" />
                        </Button>
                    </div>

                    {isAdmin && (
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                                <Button className="rounded-full bg-[#7c3aed] hover:bg-[#7c3aed]/90">
                                    <PlusCircle className="w-4 h-4 mr-2" /> {t('services.addService')}
                                </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                                <DropdownMenuItem onClick={() => {
                                    setTimeout(() => {
                                        setBatchImportModalOpen(true);
                                    }, 50);
                                }}>
                                    <Upload className="w-4 h-4 mr-2" /> {t('services.batchImport')}
                                </DropdownMenuItem>
                                <DropdownMenuItem onClick={() => navigate('/market')}>
                                    <Search className="w-4 h-4 mr-2" /> {t('services.installFromMarket')}
                                </DropdownMenuItem>
                                <DropdownMenuItem onClick={() => {
                                    setTimeout(() => {
                                        setCustomServiceModalOpen(true);
                                    }, 50);
                                }}>
                                    <Plus className="w-4 h-4 mr-2" /> {t('services.customInstall')}
                                </DropdownMenuItem>
                            </DropdownMenuContent>
                        </DropdownMenu>
                    )}
                </div>
            </div>

            <Tabs defaultValue="all" className="mb-8">
                <TabsList className="w-full max-w-3xl grid grid-cols-3 bg-muted/80 p-1 rounded-lg">
                    <TabsTrigger value="all" className="rounded-md">{t('services.allServices')}</TabsTrigger>
                    <TabsTrigger value="active" className="rounded-md">{t('services.active')}</TabsTrigger>
                    <TabsTrigger value="inactive" className="rounded-md">{t('services.inactive')}</TabsTrigger>
                </TabsList>
                <TabsContent value="all">
                    {viewMode === 'grid' ? (
                        <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 mt-6">
                            {renderGridView(allServices)}
                        </div>
                    ) : (
                        <div className="mt-6">
                            {renderListView(allServices)}
                        </div>
                    )}
                </TabsContent>
                <TabsContent value="active">
                    {viewMode === 'grid' ? (
                        <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 mt-6">
                            {renderGridView(activeServices)}
                        </div>
                    ) : (
                        <div className="mt-6">
                            {renderListView(activeServices)}
                        </div>
                    )}
                </TabsContent>
                <TabsContent value="inactive">
                    {viewMode === 'grid' ? (
                        <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 mt-6">
                            {renderGridView(inactiveServices)}
                        </div>
                    ) : (
                        <div className="mt-6">
                            {renderListView(inactiveServices)}
                        </div>
                    )}
                </TabsContent>
            </Tabs>

            {selectedService && (
                <ServiceConfigModal
                    open={configModalOpen}
                    onClose={() => setConfigModalOpen(false)}
                    service={selectedService}
                />
            )}

            {serviceToEdit && (
                <EditServiceModal
                    open={editModalOpen}
                    onClose={() => setEditModalOpen(false)}
                    service={serviceToEdit}
                    onUpdateService={handleUpdateService}
                />
            )}

            <CustomServiceModal
                open={customServiceModalOpen}
                onClose={() => setCustomServiceModalOpen(false)}
                onCreateService={handleCreateCustomService}
                autoFillEnv={autoFillEnv}
                setAutoFillEnv={setAutoFillEnv}
            />

            <BatchImportModal
                open={batchImportModalOpen}
                onClose={() => setBatchImportModalOpen(false)}
                onImportSuccess={() => {
                    fetchInstalledServices();
                    toast({
                        title: t('batchImport.importComplete'),
                        description: t('batchImport.successMessage'),
                        variant: 'default'
                    });
                }}
            />

            <ConfirmDialog
                isOpen={uninstallDialogOpen}
                onOpenChange={setUninstallDialogOpen}
                title={t('services.confirmUninstall')}
                description={t('services.confirmUninstallDescription')}
                confirmText={t('services.uninstall')}
                cancelText={t('services.cancel')}
                onConfirm={handleUninstallConfirm}
                confirmButtonVariant="destructive"
            />
        </div>
    );
} 
