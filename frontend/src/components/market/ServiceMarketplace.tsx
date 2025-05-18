import { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Search, Package, Star, Download, Filter } from 'lucide-react';
import { useMarketStore, ServiceType } from '@/store/marketStore';
import ServiceCard from './ServiceCard';
import EnvVarInputModal from './EnvVarInputModal';
import { useToast } from '@/hooks/use-toast';

export function ServiceMarketplace({ onSelectService }: { onSelectService: (serviceId: string) => void }) {

    // 使用 Zustand store
    const {
        searchTerm,
        searchResults,
        isSearching,
        activeTab,
        installedServices,
        setSearchTerm,
        setActiveTab,
        searchServices,
        fetchInstalledServices,
        installService,
        updateInstallStatus
    } = useMarketStore();

    const { toast } = useToast();

    // 新增：环境变量 Modal 相关 state
    const [envModalVisible, setEnvModalVisible] = useState(false);
    const [missingVars, setMissingVars] = useState<string[]>([]);
    const [pendingServiceId, setPendingServiceId] = useState<string | null>(null);
    const [pendingEnvVars, setPendingEnvVars] = useState<Record<string, string>>({});

    // 初始化加载
    useEffect(() => {
        if (activeTab === 'installed') {
            fetchInstalledServices();
        } else {
            searchServices();
        }
    }, [activeTab, searchServices, fetchInstalledServices]);

    // 处理标签页切换
    const handleTabChange = (value: string) => {
        setActiveTab(value as 'all' | 'npm' | 'pypi' | 'recommended' | 'installed');
        searchServices();
    };

    // 处理搜索框按下回车
    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            searchServices();
        }
    };

    // 安装服务处理函数
    const handleInstallService = async (serviceId: string, extraEnvVars: Record<string, string> = {}) => {
        setPendingServiceId(serviceId);
        let envVars = { ...extraEnvVars };
        while (true) {
            const response = await installService(serviceId, envVars);
            if (response && response.data && Array.isArray(response.data.required_env_vars) && response.data.required_env_vars.length > 0) {
                setMissingVars(response.data.required_env_vars);
                setEnvModalVisible(true);
                setPendingEnvVars(envVars);
                return; // 等待用户输入
            }
            // 安装成功或已提交任务，关闭 Modal 并重置
            setEnvModalVisible(false);
            setMissingVars([]);
            setPendingEnvVars({});
            setPendingServiceId(null);
            break;
        }
    };

    // Modal 提交回调
    const handleEnvModalSubmit = (userInputVars: Record<string, string>) => {
        if (!pendingServiceId) return;
        const merged = { ...pendingEnvVars, ...userInputVars };
        setEnvModalVisible(false);
        setMissingVars([]);
        updateInstallStatus(pendingServiceId, 'idle');
        handleInstallService(pendingServiceId, merged);
    };

    // Modal 取消回调
    const handleEnvModalCancel = () => {
        setEnvModalVisible(false);
        setMissingVars([]);
        setPendingEnvVars({});
        if (pendingServiceId) {
            // 关键：重置 installTasks 状态为 idle
            updateInstallStatus(pendingServiceId, 'idle');
        }
        setPendingServiceId(null);
    };

    // 将当前显示的服务列表计算出来
    const displayedServices = activeTab === 'installed' ? installedServices : searchResults;

    return (
        <div className="flex-1 space-y-6">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h2 className="text-3xl font-bold tracking-tight">Service Marketplace</h2>
                    <p className="text-muted-foreground mt-1">
                        Discover and install MCP services from various sources
                    </p>
                </div>
            </div>

            {/* 搜索和过滤部分 */}
            <div className="flex flex-col md:flex-row gap-4 mb-6">
                <div className="relative flex-grow">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                        className="pl-10 bg-muted/40 w-full"
                        placeholder="Search for MCP services..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        onKeyDown={handleKeyDown}
                    />
                </div>
                <Button onClick={() => searchServices()} disabled={isSearching}>
                    {isSearching ? 'Searching...' : 'Search'}
                </Button>
                <Button variant="outline" className="flex items-center gap-2">
                    <Filter className="h-4 w-4" /> Filter
                </Button>
            </div>

            {/* 选项卡分类 */}
            <Tabs value={activeTab} onValueChange={handleTabChange} className="mb-6">
                <TabsList className="w-full max-w-lg grid grid-cols-5 gap-4">
                    <TabsTrigger value="all" className="px-4">All</TabsTrigger>
                    <TabsTrigger value="npm" className="px-4">NPM</TabsTrigger>
                    <TabsTrigger value="pypi" className="px-4">PyPI</TabsTrigger>
                    <TabsTrigger value="recommended" className="px-4">Recommended</TabsTrigger>
                    <TabsTrigger value="installed" className="px-4">Installed</TabsTrigger>
                </TabsList>

                <TabsContent value="all" className="mt-6">
                    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
                        {displayedServices.map(service => (
                            <ServiceCard
                                key={service.id}
                                service={service}
                                onSelect={onSelectService}
                                onInstall={handleInstallService}
                            />
                        ))}
                        {isSearching && (
                            <div className="col-span-3 text-center py-8">
                                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
                                <p className="mt-4 text-muted-foreground">Searching for services...</p>
                            </div>
                        )}
                        {!isSearching && displayedServices.length === 0 && (
                            <div className="col-span-3 text-center py-8 text-muted-foreground">
                                <p>No services found. Try a different search term.</p>
                            </div>
                        )}
                    </div>
                </TabsContent>

                {/* 其他标签页内容类似 */}
                {['npm', 'pypi', 'recommended', 'installed'].map(tab => (
                    <TabsContent value={tab} key={tab} className="mt-6">
                        <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
                            {displayedServices
                                .filter(service => tab === 'installed' ? service.isInstalled : service.source === tab)
                                .map(service => (
                                    <ServiceCard
                                        key={service.id}
                                        service={service}
                                        onSelect={onSelectService}
                                        onInstall={handleInstallService}
                                    />
                                ))}
                            {isSearching && (
                                <div className="col-span-3 text-center py-8">
                                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
                                    <p className="mt-4 text-muted-foreground">Loading services...</p>
                                </div>
                            )}
                            {!isSearching && displayedServices.filter(service =>
                                tab === 'installed' ? service.isInstalled : service.source === tab
                            ).length === 0 && (
                                    <div className="col-span-3 text-center py-8 text-muted-foreground">
                                        <p>{`No ${tab} services available.`}</p>
                                    </div>
                                )}
                        </div>
                    </TabsContent>
                ))}
            </Tabs>
            <EnvVarInputModal
                open={envModalVisible}
                missingVars={missingVars}
                onSubmit={handleEnvModalSubmit}
                onCancel={handleEnvModalCancel}
            />
        </div>
    );
} 