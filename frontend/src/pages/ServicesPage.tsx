import { useEffect, useState, useRef } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Search, BarChart, User, PlusCircle, Trash2, Settings, Plus } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { useNavigate } from 'react-router-dom';
import { useMarketStore, ServiceType } from '@/store/marketStore';
import ServiceConfigModal from '@/components/market/ServiceConfigModal';
import CustomServiceModal, { CustomServiceData } from '@/components/market/CustomServiceModal';
import api, { APIResponse } from '@/utils/api';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';

export function ServicesPage() {
    const { toast } = useToast();
    const navigate = useNavigate();
    const { installedServices: globalInstalledServices, fetchInstalledServices, uninstallService } = useMarketStore();
    const [localInstalledServices, setLocalInstalledServices] = useState<ServiceType[]>([]);
    const [configModalOpen, setConfigModalOpen] = useState(false);
    const [customServiceModalOpen, setCustomServiceModalOpen] = useState(false);
    const [selectedService, setSelectedService] = useState<ServiceType | null>(null);
    const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
    const [pendingUninstallId, setPendingUninstallId] = useState<string | null>(null);

    const hasFetched = useRef(false);

    useEffect(() => {
        if (!hasFetched.current) {
            fetchInstalledServices();
            hasFetched.current = true;
        }
    }, [fetchInstalledServices]);

    useEffect(() => {
        setLocalInstalledServices(globalInstalledServices);
    }, [globalInstalledServices]);

    const allServices = localInstalledServices;
    const activeServices = localInstalledServices.filter(s => s.health_status === 'active' || s.health_status === 'Active');
    const inactiveServices = localInstalledServices.filter(s => s.health_status === 'inactive' || s.health_status === 'Inactive');

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
        const serviceToUninstallId = pendingUninstallId;
        setUninstallDialogOpen(false);
        setPendingUninstallId(null);

        try {
            await uninstallService(serviceToUninstallId);
            toast({
                title: 'Uninstall Complete',
                description: 'Service has been successfully uninstalled.'
            });
            setLocalInstalledServices(prev => prev.filter(s => s.id !== serviceToUninstallId));
        } catch (e: any) {
            toast({
                title: 'Uninstall Failed',
                description: e?.message || 'Unknown error',
                variant: 'destructive'
            });
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
                res = await api.post('/api/mcp_market/install_or_add_service', payload) as APIResponse<any>;

            } else {
                // For 'sse' and 'streamableHttp'
                res = await api.post('/api/mcp_market/custom_service', serviceData) as APIResponse<any>;
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
                            <Card key={service.id} className="border-border shadow-sm hover:shadow transition-shadow duration-200 bg-card/30">
                                <CardHeader>
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center">
                                            <div className="bg-primary/10 p-2 rounded-md mr-3">
                                                <Search className="w-6 h-6 text-primary" />
                                            </div>
                                            <div>
                                                <CardTitle className="text-lg">{service.display_name || service.name}</CardTitle>
                                                <CardDescription>
                                                    <span className={`inline-flex items-center px-2 py-1 text-xs rounded-full ${service.health_status === "active" || service.health_status === "Active"
                                                        ? "bg-green-100 text-green-800"
                                                        : "bg-gray-100 text-gray-800"
                                                        }`}>
                                                        {service.health_status || 'Unknown'}
                                                    </span>
                                                </CardDescription>
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
                                <CardContent>
                                    <p className="text-sm text-muted-foreground">{service.description}</p>
                                </CardContent>
                                <CardFooter className="flex justify-between">
                                    <Button variant="outline" size="sm" onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}>Configure</Button>
                                    <Button
                                        variant={service.enabled ? "outline" : "default"}
                                        size="sm"
                                        onClick={() => toast({
                                            title: `Service ${service.enabled ? "Disabled" : "Enabled"}`,
                                            description: `${service.display_name || service.name} is now ${service.enabled ? "inactive" : "active"}.`
                                        })}
                                    >
                                        {service.enabled ? "Disable" : "Enable"}
                                    </Button>
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
                            <Card key={service.id} className="border-border shadow-sm hover:shadow transition-shadow duration-200 bg-card/30">
                                <CardHeader>
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center">
                                            <div className="bg-primary/10 p-2 rounded-md mr-3">
                                                <Search className="w-6 h-6 text-primary" />
                                            </div>
                                            <div>
                                                <CardTitle className="text-lg">{service.display_name || service.name}</CardTitle>
                                                <CardDescription>
                                                    <span className="inline-flex items-center px-2 py-1 text-xs rounded-full bg-green-100 text-green-800">
                                                        Active
                                                    </span>
                                                </CardDescription>
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
                                <CardContent>
                                    <p className="text-sm text-muted-foreground">{(service as any).service_description || service.description}</p>
                                </CardContent>
                                <CardFooter className="flex justify-between">
                                    <Button variant="outline" size="sm" onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}>Configure</Button>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={() => toast({
                                            title: `Service Disabled`,
                                            description: `${(service as any).display_name || service.name} is now inactive.`
                                        })}
                                    >
                                        Disable
                                    </Button>
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
                            <Card key={service.id} className="border-border shadow-sm hover:shadow transition-shadow duration-200 bg-card/30">
                                <CardHeader>
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center">
                                            <div className="bg-primary/10 p-2 rounded-md mr-3">
                                                <Search className="w-6 h-6 text-primary" />
                                            </div>
                                            <div>
                                                <CardTitle className="text-lg">{service.display_name || service.name}</CardTitle>
                                                <CardDescription>
                                                    <span className="inline-flex items-center px-2 py-1 text-xs rounded-full bg-gray-100 text-gray-800">
                                                        Inactive
                                                    </span>
                                                </CardDescription>
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
                                <CardContent>
                                    <p className="text-sm text-muted-foreground">{(service as any).service_description || service.description}</p>
                                </CardContent>
                                <CardFooter className="flex justify-between">
                                    <Button variant="outline" size="sm" onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}>Configure</Button>
                                    <Button
                                        variant="default"
                                        size="sm"
                                        onClick={() => toast({
                                            title: `Service Enabled`,
                                            description: `${(service as any).display_name || service.name} is now active.`
                                        })}
                                    >
                                        Enable
                                    </Button>
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

            <div className="mt-12">
                <h3 className="text-2xl font-bold mb-4">Usage Statistics</h3>
                <Card className="shadow-sm border bg-card/30">
                    <CardHeader>
                        <CardTitle>Service Utilization</CardTitle>
                        <CardDescription>Performance overview for the last 30 days</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <Table>
                            <TableCaption>A summary of your service usage.</TableCaption>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>Service</TableHead>
                                    <TableHead>Requests</TableHead>
                                    <TableHead>Success Rate</TableHead>
                                    <TableHead className="text-right">Avg. Latency</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {/* 这里可后续对接真实统计数据 */}
                            </TableBody>
                        </Table>
                    </CardContent>
                </Card>
            </div>
        </div>
    );
} 