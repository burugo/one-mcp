import { useEffect, useState, useRef } from 'react';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableHead, TableHeader, TableRow, TableCell } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Search, PlusCircle, Trash2, Plus, RotateCcw, Grid, List } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { useNavigate } from 'react-router-dom';
import { useMarketStore, ServiceType } from '@/store/marketStore';
import ServiceConfigModal from '@/components/market/ServiceConfigModal';
import CustomServiceModal, { CustomServiceData } from '@/components/market/CustomServiceModal';
import api, { APIResponse } from '@/utils/api';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { useTranslation } from 'react-i18next';
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
    const { installedServices: globalInstalledServices, fetchInstalledServices, uninstallService, toggleService, checkServiceHealth } = useMarketStore();
    const [configModalOpen, setConfigModalOpen] = useState(false);
    const [customServiceModalOpen, setCustomServiceModalOpen] = useState(false);
    const [selectedService, setSelectedService] = useState<ServiceType | null>(null);
    const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
    const [pendingUninstallId, setPendingUninstallId] = useState<string | null>(null);
    const [togglingServices, setTogglingServices] = useState<Set<string>>(new Set());
    const [checkingHealthServices, setCheckingHealthServices] = useState<Set<string>>(new Set());
    const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

    const hasFetched = useRef(false);

    useEffect(() => {
        if (!hasFetched.current) {
            fetchInstalledServices();
            hasFetched.current = true;
        }
    }, [fetchInstalledServices]);

    const allServices = globalInstalledServices;
    const activeServices = globalInstalledServices.filter(s => s.enabled === true);
    const inactiveServices = globalInstalledServices.filter(s => s.enabled === false);

    const handleSaveVar = async (varName: string, value: string) => {
        if (!selectedService) return;
        const service_id = selectedService.id;
        const res = await api.patch('/mcp_market/env_var', {
            service_id,
            var_name: varName,
            var_value: value,
        }) as APIResponse<any>;
        if (res.success) {
            toast({
                title: 'Saved',
                description: `${varName} has been saved.`
            });
            fetchInstalledServices();
        } else {
            throw new Error(res.message || 'Failed to save');
        }
    };

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
        return envStr.split('\\n').reduce((acc, line) => {
            const [key, ...valueParts] = line.split('=');
            if (key?.trim() && valueParts.length > 0) {
                acc[key.trim()] = valueParts.join('=').trim();
            }
            return acc;
        }, {} as Record<string, string>);
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

    const handleCreateCustomService = async (serviceData: CustomServiceData) => {
        try {
            let res;
            if (serviceData.type === 'stdio') {
                let packageName = '';
                let packageManager = '';
                const commandParts = serviceData.command?.split(' ');
                if (commandParts && commandParts.length > 1) {
                    if (commandParts[0] === 'npx') {
                        packageManager = 'npm';
                        packageName = commandParts.slice(1).join(' '); // Allow package names with spaces, though uncommon
                    } else if (commandParts[0] === 'uvx') {
                        packageManager = 'uv'; // Or 'pypi', 'pip' based on exact backend expectation for uvx
                        packageName = commandParts.slice(1).join(' ');
                    }
                }

                if (!packageManager || !packageName) {
                    throw new Error('无法从命令中解析包管理器或包名称。命令必须以 "npx " 或 "uvx " 开头。');
                }

                // Arguments from serviceData.arguments are currently not passed to install_or_add_service
                // as the backend handler InstallOrAddService typically auto-generates ArgsJSON.
                // If custom arguments are essential here, InstallOrAddService would need modification.

                const payload = {
                    source_type: 'marketplace', // Or another suitable type if backend expects something specific for custom stdio
                    package_name: packageName,
                    package_manager: packageManager,
                    display_name: serviceData.name,
                    user_provided_env_vars: parseEnvironments(serviceData.environments),
                    // version: 'latest', // Optional: InstallOrAddService might need a version
                    // service_description: `Custom stdio service: ${serviceData.name}`, // Optional
                    // category: 'utility', // Optional
                    // headers: {}, // Not applicable for stdio
                };
                res = await api.post('/mcp_market/install_or_add_service', payload) as APIResponse<any>;

            } else {
                // For 'sse' and 'streamableHttp'
                res = await api.post('/mcp_market/custom_service', serviceData) as APIResponse<any>;
            }

            if (res.success) {
                toast({
                    title: '创建成功',
                    description: `服务 ${serviceData.name} 已成功创建/提交安装`
                });
                fetchInstalledServices(); // Refresh the list
                // If res.data contains the new service, could potentially return it
                return res.data;
            } else {
                throw new Error(res.message || '创建失败');
            }
        } catch (error: any) {
            toast({
                title: '创建失败',
                description: error.message || '未知错误',
                variant: 'destructive'
            });
            throw error; // Re-throw to be caught by the modal if necessary
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
                                    <button
                                        className="p-1 rounded hover:bg-red-100 text-red-500"
                                        onClick={() => handleUninstallClick(service.id)}
                                        title={t('services.uninstallService')}
                                    >
                                        <Trash2 size={16} />
                                    </button>
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
                        <button
                            className="ml-2 p-1 rounded hover:bg-red-100 text-red-500"
                            onClick={() => handleUninstallClick(service.id)}
                            title={t('services.uninstallService')}
                        >
                            <Trash2 size={18} />
                        </button>
                    </div>
                </CardHeader>
                <CardContent className="flex-grow">
                    <p className="text-sm text-muted-foreground line-clamp-2 mb-3">{service.description}</p>
                    {/* RPD Limit and Usage Display */}
                    {service.rpd_limit && service.rpd_limit > 0 && (
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

                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button className="rounded-full bg-[#7c3aed] hover:bg-[#7c3aed]/90">
                                <PlusCircle className="w-4 h-4 mr-2" /> {t('services.addService')}
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
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
                    onSaveVar={handleSaveVar}
                />
            )}

            <CustomServiceModal
                open={customServiceModalOpen}
                onClose={() => setCustomServiceModalOpen(false)}
                onCreateService={handleCreateCustomService}
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