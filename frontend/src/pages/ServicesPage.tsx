import { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Search, BarChart, User, PlusCircle, Trash2 } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { useNavigate } from 'react-router-dom';
import { useMarketStore } from '@/store/marketStore';
import ServiceConfigModal from '@/components/market/ServiceConfigModal';
import api from '@/utils/api';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';

export function ServicesPage() {
    const { toast } = useToast();
    const navigate = useNavigate();
    const { installedServices, fetchInstalledServices, restartService, uninstallService, setInstalledServices } = useMarketStore();
    const [configModalOpen, setConfigModalOpen] = useState(false);
    const [selectedService, setSelectedService] = useState<any>(null);
    const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
    const [pendingUninstallId, setPendingUninstallId] = useState<string | null>(null);
    const [removingIds, setRemovingIds] = useState<string[]>([]);

    useEffect(() => {
        fetchInstalledServices().then(() => {
            setInstalledServices(useMarketStore.getState().installedServices);
        });
    }, [fetchInstalledServices]);

    // 过滤逻辑
    const allServices = installedServices;
    const activeServices = installedServices.filter(s => s.health_status === 'active' || s.health_status === 'Active');
    const inactiveServices = installedServices.filter(s => s.health_status === 'inactive' || s.health_status === 'Inactive');

    // 保存单个环境变量
    const handleSaveVar = async (varName: string, value: string) => {
        if (!selectedService) return;
        const service_id = selectedService.id;
        const res = await api.patch('/mcp_market/env_var', {
            service_id,
            var_name: varName,
            var_value: value,
        });
        if (res.success) {
            toast({ title: 'Saved', description: `${varName} 已保存`, variant: 'success' });
            fetchInstalledServices(); // 刷新服务数据
        } else {
            throw new Error(res.message || '保存失败');
        }
    };

    const handleUninstallClick = (serviceId: string) => {
        setPendingUninstallId(serviceId);
        setUninstallDialogOpen(true);
    };
    const handleUninstallConfirm = async () => {
        if (!pendingUninstallId) return;
        setUninstallDialogOpen(false);
        try {
            await uninstallService(pendingUninstallId);
            toast({ title: '卸载成功', description: '服务已成功卸载', variant: 'success' });
            // 本地立即移除卡片
            setInstalledServices(prev => prev.filter(s => s.id !== pendingUninstallId));
        } catch (e: any) {
            toast({ title: '卸载失败', description: e?.message || '未知错误', variant: 'destructive' });
        }
        setPendingUninstallId(null);
    };

    return (
        <div className="w-full space-y-8">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h2 className="text-3xl font-bold tracking-tight">MCP Services</h2>
                    <p className="text-muted-foreground mt-1">Manage and configure your multi-cloud platform services</p>
                </div>
                <Button onClick={() => navigate('/market')} className="rounded-full bg-[#7c3aed] hover:bg-[#7c3aed]/90">
                    <PlusCircle className="w-4 h-4 mr-2" /> Add Service
                </Button>
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
                        ) : allServices.filter(s => !removingIds.includes(s.id)).map(service => (
                            <Card key={service.id} className="border-border shadow-sm hover:shadow transition-shadow duration-200 bg-card/30">
                                <CardHeader>
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center">
                                            <div className="bg-primary/10 p-2 rounded-md mr-3">
                                                {/* 可根据 service.Icon 字段渲染图标 */}
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
                        ) : activeServices.filter(s => !removingIds.includes(s.id)).map(service => (
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
                                    <p className="text-sm text-muted-foreground">{service.service_description || service.description}</p>
                                </CardContent>
                                <CardFooter className="flex justify-between">
                                    <Button variant="outline" size="sm" onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}>Configure</Button>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={() => toast({
                                            title: `Service Disabled`,
                                            description: `${service.display_name || service.name} is now inactive.`
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
                        ) : inactiveServices.filter(s => !removingIds.includes(s.id)).map(service => (
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
                                    <p className="text-sm text-muted-foreground">{service.service_description || service.description}</p>
                                </CardContent>
                                <CardFooter className="flex justify-between">
                                    <Button variant="outline" size="sm" onClick={() => { setSelectedService(service); setConfigModalOpen(true); }}>Configure</Button>
                                    <Button
                                        variant="default"
                                        size="sm"
                                        onClick={() => toast({
                                            title: `Service Enabled`,
                                            description: `${service.display_name || service.name} is now active.`
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

            <ServiceConfigModal
                open={configModalOpen}
                service={selectedService}
                onClose={() => setConfigModalOpen(false)}
                onSaveVar={handleSaveVar}
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

            <ConfirmDialog
                isOpen={uninstallDialogOpen}
                onOpenChange={setUninstallDialogOpen}
                title="确认卸载"
                description="确定要卸载该服务吗？此操作不可撤销。"
                confirmText="卸载"
                cancelText="取消"
                confirmButtonVariant="destructive"
                onConfirm={handleUninstallConfirm}
            />
        </div>
    );
} 