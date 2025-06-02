import { useEffect, useState, useRef } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCaption, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Search, PlusCircle, Trash2, Plus, RotateCcw } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { useNavigate } from 'react-router-dom';
import { useMarketStore, ServiceType } from '@/store/marketStore';
import ServiceConfigModal from '@/components/market/ServiceConfigModal';
import CustomServiceModal, { CustomServiceData } from '@/components/market/CustomServiceModal';
import api, { APIResponse } from '@/utils/api';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { Switch } from '@/components/ui/switch';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';

export function ServicesPage() {
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

    return (
        <div className="w-full space-y-8">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h2 className="text-3xl font-bold tracking-tight">MCP Services</h2>
                    <p className="text-muted-foreground mt-1">Manage and configure your multi-cloud platform services</p>
                </div>
                <div className="flex space-x-2">
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button className="rounded-full bg-[#7c3aed] hover:bg-[#7c3aed]/90">
                                <PlusCircle className="w-4 h-4 mr-2" /> Add Service
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => navigate('/market')}>
                                <Search className="w-4 h-4 mr-2" /> 从市场安装
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => {
                                setTimeout(() => {
                                    setCustomServiceModalOpen(true);
                                }, 50);
                            }}>
                                <Plus className="w-4 h-4 mr-2" /> 自定义安装
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </div>

            <Tabs defaultValue="all" className="mb-8">
                <TabsList className="w-full max-w-3xl grid grid-cols-3 bg-muted/80 p-1 rounded-lg">
                    <TabsTrigger value="all" className="rounded-md">All Services</TabsTrigger>
                    <TabsTrigger value="active" className="rounded-md">Active</TabsTrigger>
                    <TabsTrigger value="inactive" className="rounded-md">Inactive</TabsTrigger>
                </TabsList>
                <TabsContent value="all">
                    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 mt-6">
                        {allServices.length === 0 ? (
                            <div className="col-span-3 text-center py-8 text-muted-foreground">
                                <p>No installed services.</p>
                            </div>
                        ) : allServices.map(service => (
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
                                                        title="刷新健康状态"
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
                                            title="卸载服务"
                                        >
                                            <Trash2 size={18} />
                                        </button>
                                    </div>
                                </CardHeader>
                                <CardContent className="flex-grow">
                                    <p className="text-sm text-muted-foreground line-clamp-2">{service.description}</p>
                                </CardContent>
                                <CardFooter className="flex justify-between items-end mt-auto">
                                    <Button variant="outline" size="sm" className="h-6" onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}>Configure</Button>
                                    <Switch
                                        checked={service.enabled || false}
                                        onCheckedChange={() => handleToggleService(service.id)}
                                        disabled={togglingServices.has(service.id)}
                                    />
                                </CardFooter>
                            </Card>
                        ))}
                    </div>
                </TabsContent>
                <TabsContent value="active">
                    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 mt-6">
                        {activeServices.length === 0 ? (
                            <div className="col-span-3 text-center py-8 text-muted-foreground">
                                <p>No active services.</p>
                            </div>
                        ) : activeServices.map(service => (
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
                                                        title="刷新健康状态"
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
                                            title="卸载服务"
                                        >
                                            <Trash2 size={18} />
                                        </button>
                                    </div>
                                </CardHeader>
                                <CardContent className="flex-grow">
                                    <p className="text-sm text-muted-foreground line-clamp-2">{(service as any).service_description || service.description}</p>
                                </CardContent>
                                <CardFooter className="flex justify-between items-end mt-auto">
                                    <Button variant="outline" size="sm" className="h-6" onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}>Configure</Button>
                                    <Switch
                                        checked={service.enabled || false}
                                        onCheckedChange={() => handleToggleService(service.id)}
                                        disabled={togglingServices.has(service.id)}
                                    />
                                </CardFooter>
                            </Card>
                        ))}
                    </div>
                </TabsContent>
                <TabsContent value="inactive">
                    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 mt-6">
                        {inactiveServices.length === 0 ? (
                            <div className="col-span-3 text-center py-8 text-muted-foreground">
                                <p>No inactive services.</p>
                            </div>
                        ) : inactiveServices.map(service => (
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
                                                        title="刷新健康状态"
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
                                            title="卸载服务"
                                        >
                                            <Trash2 size={18} />
                                        </button>
                                    </div>
                                </CardHeader>
                                <CardContent className="flex-grow">
                                    <p className="text-sm text-muted-foreground line-clamp-2">{(service as any).service_description || service.description}</p>
                                </CardContent>
                                <CardFooter className="flex justify-between items-end mt-auto">
                                    <Button variant="outline" size="sm" className="h-6" onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}>Configure</Button>
                                    <Switch
                                        checked={service.enabled || false}
                                        onCheckedChange={() => handleToggleService(service.id)}
                                        disabled={togglingServices.has(service.id)}
                                    />
                                </CardFooter>
                            </Card>
                        ))}
                    </div>
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
                title="确认卸载"
                description="确定要卸载此服务吗？这将移除所有相关配置。"
                confirmText="卸载"
                cancelText="取消"
                onConfirm={handleUninstallConfirm}
                confirmButtonVariant="destructive"
            />
        </div>
    );
} 