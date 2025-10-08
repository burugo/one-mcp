import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Search, CheckCircle, XCircle } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { useMarketStore, ServiceType /* , MarketSource */ } from '@/store/marketStore';
import ServiceCard from './ServiceCard';
import EnvVarInputModal from './EnvVarInputModal';
import { useToast } from '@/hooks/use-toast';

import { useTranslation } from 'react-i18next';

export function ServiceMarketplace({ onSelectService }: { onSelectService: (serviceId: string) => void }) {
    const { t } = useTranslation();

    // 使用 Zustand store
    const {
        searchTerm,
        searchResults,
        isSearching,
        setSearchTerm,
        searchServices,
        fetchInstalledServices,
        installService,
        updateInstallStatus,
        installTasks,
        // 暂时注释掉 tab 相关状态，以后添加 PyPI 时恢复
        // activeMarketTab,
        // setActiveMarketTab
    } = useMarketStore();

    const { toast } = useToast();

    // 新增：环境变量 Modal 相关 state
    const [envModalVisible, setEnvModalVisible] = useState(false);
    const [missingVars, setMissingVars] = useState<string[]>([]);
    const [pendingServiceId, setPendingServiceId] = useState<string | null>(null);
    const [pendingEnvVars, setPendingEnvVars] = useState<Record<string, string>>({});

    // 新增：安装进度对话框相关 state
    const [showInstallDialog, setShowInstallDialog] = useState(false);
    const [currentInstallingService, setCurrentInstallingService] = useState<ServiceType | null>(null);

    // Effect to fetch services when activeMarketTab changes or on initial load. --> Changed to: Effect for initial load only.
    useEffect(() => {
        // This will be called on initial load.
        // searchServices in the store uses the initial activeMarketTab ('npm') 
        // and initial searchTerm (empty), which clears results.
        // This provides an empty state for the initially active tab, awaiting user search.
        searchServices();
    }, [searchServices]); // Now only depends on stable searchServices, so runs once on mount.

    // 处理搜索框按下回车
    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            searchServices();
        }
    };

    // 简化的安装服务处理函数，保持原有流程
    const handleInstallService = async (serviceId: string, extraEnvVars: Record<string, string> = {}) => {
        // 找到对应的服务
        const service = searchResults.find(s => s.id === serviceId);
        if (!service) {
            toast({
                variant: "destructive",
                title: "Service Not Found",
                description: "Could not find the service to install."
            });
            return;
        }

        setPendingServiceId(serviceId);
        setCurrentInstallingService(service);
        const envVars = { ...extraEnvVars };

        while (true) {
            const response = await installService(serviceId, envVars);
            if (response && response.data && Array.isArray(response.data.required_env_vars) && response.data.required_env_vars.length > 0) {
                setMissingVars(response.data.required_env_vars);
                setEnvModalVisible(true);
                setPendingEnvVars(envVars);
                return; // 等待用户输入
            }
            // 安装成功或已提交任务，显示进度对话框
            setEnvModalVisible(false);
            setShowInstallDialog(true);
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
            updateInstallStatus(pendingServiceId, 'idle');
        }
        setPendingServiceId(null);
        setCurrentInstallingService(null);
    };

    // 关闭安装对话框
    const closeInstallDialog = () => {
        const installTask = currentInstallingService ? installTasks[currentInstallingService.id] : undefined;
        setShowInstallDialog(false);

        // 如果安装成功，刷新已安装服务列表
        if (installTask?.status === 'success') {
            fetchInstalledServices();
            toast({
                title: "Installation Successful",
                description: `${currentInstallingService?.name} has been installed and is ready to use.`
            });
        }

        // 当安装仍在进行时，保持安装任务继续轮询，但允许用户关闭对话框
        if (installTask?.status !== 'installing') {
            setCurrentInstallingService(null);
            setPendingServiceId(null);
        }
    };

    // 将当前显示的服务列表计算出来
    const displayedServices = searchResults;

    // 获取当前安装任务
    const currentInstallTask = currentInstallingService ? installTasks[currentInstallingService.id] : undefined;

    return (
        <div className="flex-1 space-y-6">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h2 className="text-3xl font-bold tracking-tight">{t('serviceMarketplace.title')}</h2>
                    <p className="text-muted-foreground mt-1">
                        {t('serviceMarketplace.description')}
                    </p>
                </div>
            </div>

            {/* 搜索和过滤部分 */}
            <div className="flex flex-col md:flex-row gap-4 mb-6">
                <div className="relative flex-grow">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                        className="pl-10 bg-muted/40 w-full"
                        placeholder={t('serviceMarketplace.searchPlaceholder')}
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        onKeyDown={handleKeyDown}
                    />
                </div>
                <Button onClick={() => searchServices()} disabled={isSearching}>
                    {isSearching ? 'Searching...' : t('serviceMarketplace.searchButton')}
                </Button>
            </div>

            {/* 服务列表 - 暂时移除 Tabs，以后添加 PyPI 市场时再恢复 */}
            {/* TODO: 将来需要添加 PyPI 市场时，恢复以下 Tabs 结构 */}
            {/* 
            <Tabs value={activeMarketTab} onValueChange={(value) => setActiveMarketTab(value as MarketSource)} className="mb-6">
                <TabsList className="w-full max-w-lg grid grid-cols-2 gap-4">
                    <TabsTrigger value="npm" className="px-4">NPM</TabsTrigger>
                    <TabsTrigger value="pypi" className="px-4">PyPI</TabsTrigger>
                </TabsList>

                <TabsContent value="npm" className="mt-6">
                    ... NPM content ...
                </TabsContent>
                <TabsContent value="pypi" className="mt-6">
                    ... PyPI content ...
                </TabsContent>
            </Tabs>
            */}

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
                        <p className="mt-4 text-muted-foreground">{t('serviceMarketplace.searching')}</p>
                    </div>
                )}
                {!isSearching && displayedServices.length === 0 && (
                    <div className="col-span-3 text-center py-8 text-muted-foreground">
                        <p>{t('serviceMarketplace.noServicesFound')}</p>
                    </div>
                )}
            </div>

            {/* 环境变量输入模态框 */}
            <EnvVarInputModal
                open={envModalVisible}
                missingVars={missingVars}
                onSubmit={handleEnvModalSubmit}
                onCancel={handleEnvModalCancel}
            />

            {/* 安装进度对话框 */}
            <Dialog open={showInstallDialog} onOpenChange={closeInstallDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>
                            {!currentInstallTask && 'Installing Service...'}
                            {currentInstallTask?.status === 'installing' && 'Installing Service...'}
                            {currentInstallTask?.status === 'success' && 'Installation Complete'}
                            {currentInstallTask?.status === 'error' && 'Installation Failed'}
                        </DialogTitle>
                        <DialogDescription>
                            {!currentInstallTask && `Installing ${currentInstallingService?.name} from ${currentInstallingService?.source}`}
                            {currentInstallTask?.status === 'installing' && `Installing ${currentInstallingService?.name} from ${currentInstallingService?.source}`}
                            {currentInstallTask?.status === 'success' && 'The service was installed successfully'}
                            {currentInstallTask?.status === 'error' && 'There was a problem during installation'}
                        </DialogDescription>
                    </DialogHeader>

                    <div className="my-4">
                        <div className="bg-muted p-4 rounded-md h-64 overflow-y-auto font-mono text-sm">
                            {currentInstallTask?.logs.map((log, index) => (
                                <div key={index} className="pb-1">
                                    <span className="text-primary">{'>'}</span> {log}
                                </div>
                            ))}
                            {(!currentInstallTask || currentInstallTask.status === 'installing') && (
                                <div className="animate-pulse">
                                    <span className="text-primary">{'>'}</span> _
                                </div>
                            )}
                        </div>
                    </div>

                    <DialogFooter className="flex items-center justify-between">
                        {(!currentInstallTask || currentInstallTask.status === 'installing') && (
                            <div className="flex items-center text-sm text-muted-foreground">
                                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary mr-2"></div>
                                {t('serviceMarketplace.installing')}
                            </div>
                        )}
                        {currentInstallTask?.status === 'success' && (
                            <div className="flex items-center text-sm text-green-500">
                                <CheckCircle className="h-4 w-4 mr-2" />
                                {t('serviceMarketplace.installationComplete')}
                            </div>
                        )}
                        {currentInstallTask?.status === 'error' && (
                            <div className="flex items-center text-sm text-red-500">
                                <XCircle className="h-4 w-4 mr-2" />
                                {t('serviceMarketplace.installationFailed')}
                            </div>
                        )}

                        <Button
                            disabled={!currentInstallTask || currentInstallTask.status === 'installing'}
                            onClick={closeInstallDialog}
                        >
                            {currentInstallTask?.status === 'success' ? 'Finish' : 'Close'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
} 
