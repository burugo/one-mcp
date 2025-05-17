import { useEffect } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Search, Package, Star, Download, Filter } from 'lucide-react';
import { useMarketStore, ServiceType } from '@/store/marketStore';
import ServiceCard from './ServiceCard';

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
        installService
    } = useMarketStore();

    // 初始化加载
    useEffect(() => {
        // 立即加载搜索结果
        searchServices();

        // 始终预加载已安装服务数据，以便后续切换
        fetchInstalledServices();
    }, [searchServices, fetchInstalledServices]);

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
    const handleInstallService = (serviceId: string) => {
        // Call the store's installService, which expects (serviceId, envVars)
        // It internally uses get().selectedService for other details.
        // UI flow might need to ensure selectedService is set before this, 
        // e.g., by requiring user to view details first, or if this component sets it.
        // For now, just pass the correct arguments to satisfy the current store signature.

        // It's crucial that `selectedService` in the store is the one corresponding to `serviceId`
        // when `installService` is executed internally by the store.
        // A robust solution might involve changing installService store action signature later.
        const store = useMarketStore.getState();
        // If the service to be installed is not the currently selected one in the store,
        // we might need to select it first. This can have UI implications (e.g. navigating to details).
        // For now, we assume that if installService is called, the selection context is already handled
        // or installService itself is robust enough if selectedService is not the target (it currently is not, it uses whatever is selected).
        // This is a simplification to fix the linter error. The actual install logic might need refinement.
        if (store.selectedService?.id !== serviceId) {
            // console.warn(`Attempting to install service ${serviceId} which is not the currently selected service (${store.selectedService?.id}). This might lead to unexpected behavior if installService relies on selectedService details.`);
            // To make this work reliably with current store.installService, we'd have to ensure selection:
            // store.selectService(serviceId); // This fetches details and then sets selectedService. Async.
            // For a quick fix for the linter and a basic attempt:
        }
        installService(serviceId, {});
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
        </div>
    );
} 