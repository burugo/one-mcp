import { useState, useEffect } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { ChevronLeft, Package, Star, Download, CheckCircle, XCircle, AlertCircle } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { useMarketStore } from '@/store/marketStore';
import EnvVarInputModal from './EnvVarInputModal';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { useTranslation } from 'react-i18next';

export function ServiceDetails({ onBack }: { onBack: () => void }) {
    const { t } = useTranslation();
    const { toast } = useToast();
    const {
        selectedService,
        isLoadingDetails,
        installTasks,
        installService,
        uninstallService,
        updateInstallStatus,
        fetchServiceDetails
    } = useMarketStore();

    // State for installation log dialog
    const [showInstallDialog, setShowInstallDialog] = useState(false);

    // State for EnvVarInputModal (similar to ServiceMarketplace)
    const [envModalVisible, setEnvModalVisible] = useState(false);
    const [missingVars, setMissingVars] = useState<string[]>([]);
    const [pendingServiceId, setPendingServiceId] = useState<string | null>(null);
    const [currentEnvVars, setCurrentEnvVars] = useState<Record<string, string>>({});
    // Reset modal states if selectedService changes
    useEffect(() => {
        setEnvModalVisible(false);
        setMissingVars([]);
        setPendingServiceId(null);
        setCurrentEnvVars({});
        setShowInstallDialog(false); // Also reset log dialog
    }, [selectedService?.id]);

    // 获取当前服务的安装任务（如果有）
    const installTask = selectedService ?
        installTasks[selectedService.id] : undefined;

    // Helper function to check if a value is a placeholder/example value
    const isPlaceholderValue = (value: string): boolean => {
        if (!value || value.trim() === '') return true;

        const lowerValue = value.toLowerCase().trim();

        // Common placeholder patterns
        const placeholderPatterns = [
            'your-',
            'enter-',
            'api-key-here',
            'api-token-here',
            'token-here',
            'key-here',
            'secret-here',
            'paste-',
            'insert-',
            'add-'
        ];

        return placeholderPatterns.some(pattern => lowerValue.includes(pattern));
    };

    // Modified to handle dynamic env var requirements
    const startInstallation = async (initialEnvVars?: Record<string, string>) => {
        if (!selectedService) return;

        const envVarsToSubmit: Record<string, string> = {};
        const missingRequiredVars: string[] = [];

        if (initialEnvVars) { // If env vars are explicitly passed (e.g., from modal)
            Object.assign(envVarsToSubmit, initialEnvVars);
        } else { // If not from modal, collect from service env vars
            (selectedService.envVars || []).forEach(env => {
                const value = env.value?.trim() || '';

                // Only include if value exists, is not empty, and is not a placeholder
                if (value && !isPlaceholderValue(value)) {
                    envVarsToSubmit[env.name] = value;
                } else if (!env.optional) {
                    // If required env var is missing or is a placeholder, add to missing list
                    missingRequiredVars.push(env.name);
                }
            });
        }

        // If there are missing required vars and we haven't already shown the modal
        if (missingRequiredVars.length > 0 && !initialEnvVars) {
            setMissingVars(missingRequiredVars);
            setCurrentEnvVars(envVarsToSubmit);
            setPendingServiceId(selectedService.id);
            setEnvModalVisible(true);
            return;
        }

        setCurrentEnvVars(envVarsToSubmit); // Store for potential re-submission if modal is needed
        setPendingServiceId(selectedService.id);

        try {
            // Call installService from the store
            const response = await installService(selectedService.id, envVarsToSubmit);

            // Check if the response indicates missing env vars (adjust based on actual response structure)
            if (response && response.data && Array.isArray(response.data.required_env_vars) && response.data.required_env_vars.length > 0) {
                setMissingVars(response.data.required_env_vars);
                setEnvModalVisible(true); // Show EnvVarInputModal
                //setShowInstallDialog(false); // Ensure log dialog is hidden if env modal is shown
            } else {
                // No missing vars or successful submission, proceed to show installation log dialog
                setEnvModalVisible(false);
                setShowInstallDialog(true);
            }
        } catch (error) {
            console.error("Installation trigger error:", error);
            toast({
                title: "Installation Error",
                description: "Failed to start installation process.",
                variant: "destructive"
            });
            // Optionally reset state here
            setPendingServiceId(null);
            setCurrentEnvVars({});
            if (selectedService) updateInstallStatus(selectedService.id, 'error', 'Failed to trigger install');
        }
    };

    const handleEnvModalSubmit = (userInputVars: Record<string, string>) => {
        if (!pendingServiceId) return;
        const mergedEnvVars = { ...currentEnvVars, ...userInputVars };
        setEnvModalVisible(false);
        // It's important to reset the status in the store so installService can be called again if needed
        // or simply attempt install again with merged vars
        updateInstallStatus(pendingServiceId, 'idle');
        startInstallation(mergedEnvVars); // Re-trigger installation with new vars
    };

    const handleEnvModalCancel = () => {
        setEnvModalVisible(false);
        if (pendingServiceId) {
            updateInstallStatus(pendingServiceId, 'idle'); // Reset status to allow re-initiation
        }
        setMissingVars([]);
        //setCurrentEnvVars({}); // Keep current env vars if user cancels
        setPendingServiceId(null);
    };

    // 关闭安装对话框
    const closeInstallDialog = async () => {
        setShowInstallDialog(false);

        // 如果安装成功，刷新当前页面状态而不是返回上一级
        if (installTask?.status === 'success' && selectedService) {
            toast({
                title: "Installation Successful",
                description: `${selectedService.name} has been installed and is ready to use.`
            });

            // 刷新当前服务详情以更新安装状态
            await fetchServiceDetails(selectedService.id, selectedService.name, selectedService.source);
        }
    };

    // 卸载服务
    const handleUninstall = async () => {
        if (!selectedService || typeof selectedService.installed_service_id !== 'number') {
            toast({
                title: "Cannot Uninstall",
                description: "Service data is incomplete or service ID is missing.",
                variant: "destructive",
            });
            return;
        }

        // 显示确认对话框
        if (window.confirm(`Are you sure you want to uninstall ${selectedService.name}?`)) {
            try {
                await uninstallService(selectedService.installed_service_id); // Use numeric ID and wait for completion
                toast({
                    title: "Service Uninstalled",
                    description: `${selectedService.name} has been uninstalled.`
                });

                // 刷新当前服务详情以更新状态
                // 移除 searchServices() 调用，依赖 uninstallService 中的乐观更新
                await fetchServiceDetails(selectedService.id, selectedService.name, selectedService.source); // 重新获取当前服务详情

            } catch (error: unknown) {
                let message = "Failed to uninstall service.";
                if (error instanceof Error) {
                    message = error.message;
                }
                toast({
                    title: "Uninstall Failed",
                    description: message,
                    variant: "destructive"
                });
            }
        }
    };

    // 加载状态
    if (isLoadingDetails) {
        return (
            <div className="flex-1 p-6 flex justify-center items-center">
                <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto"></div>
                    <p className="mt-4 text-muted-foreground">{t('serviceDetails.loadingServiceDetails')}</p>
                </div>
            </div>
        );
    }

    // 服务不存在
    if (!selectedService) {
        return (
            <div className="flex-1 p-6">
                <Button variant="ghost" onClick={onBack} className="mb-6">
                    <ChevronLeft className="mr-2 h-4 w-4" />
                    {t('serviceDetails.backToMarketplace')}
                </Button>
                <div className="text-center py-12">
                    <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                    <h3 className="text-xl font-semibold mb-2">{t('serviceDetails.serviceNotFound')}</h3>
                    <p className="text-muted-foreground">{t('serviceDetails.serviceNotFoundDescription')}</p>
                </div>
            </div>
        );
    }

    return (
        <div className="flex-1 space-y-6">
            <Button variant="ghost" onClick={onBack} className="mb-2">
                <ChevronLeft className="mr-2 h-4 w-4" />
                {t('backToMarketplace')}
            </Button>

            {/* 服务头部信息 */}
            <div className="flex flex-col md:flex-row gap-6 items-start">
                <div className="bg-primary/10 p-4 rounded-lg flex-shrink-0">
                    <Package className="h-16 w-16 text-primary" />
                </div>

                <div className="flex-grow min-w-0">
                    <h1 className="text-3xl font-bold break-words mb-2">
                        <a href={`https://www.npmjs.com/package/${selectedService.name}`} target="_blank" rel="noopener noreferrer" className="hover:underline">
                            {selectedService.name}
                        </a>
                    </h1>
                    <div className="flex flex-wrap items-center gap-4 mt-2 text-sm text-muted-foreground">
                        <div>v{selectedService.version}</div>
                        {typeof selectedService.stars === 'number' && (
                            <div className="flex items-center gap-1" title={`${selectedService.stars.toLocaleString()} Stars`}>
                                <Star className="h-3.5 w-3.5 text-yellow-400 fill-yellow-400" />
                                <span>{selectedService.stars.toLocaleString()}</span>
                            </div>
                        )}
                        {typeof selectedService.downloads === 'number' && (
                            <div className="flex items-center gap-1" title={`${selectedService.downloads.toLocaleString()} Weekly Downloads`}>
                                <Download className="h-3.5 w-3.5 text-green-500" />
                                <span>{selectedService.downloads.toLocaleString()}</span>
                            </div>
                        )}
                        <div>
                            <span>{t('by')} {typeof selectedService.author === 'string' ? selectedService.author : selectedService.author?.name || 'Unknown Author'}</span>
                        </div>
                        <div>
                            <span>{t('source')}: {selectedService.source}</span>
                        </div>
                    </div>
                    <p className="mt-4 text-balance">{selectedService.description}</p>
                </div>

                {selectedService.isInstalled ? (
                    <Button onClick={handleUninstall} variant="destructive" className="md:self-start flex-shrink-0">
                        {t('uninstallService')}
                    </Button>
                ) : (
                    <Button onClick={() => startInstallation()} className="md:self-start flex-shrink-0">
                        {t('installService')}
                    </Button>
                )}
            </div>

            {/* README 内容 */}
            <div className="mt-8">
                <Card>
                    <CardContent className="pt-6">
                        <div className="prose dark:prose-invert max-w-none">
                            <ReactMarkdown remarkPlugins={[remarkGfm]}>
                                {selectedService.readme}
                            </ReactMarkdown>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* 安装进度对话框 */}
            <Dialog open={showInstallDialog} onOpenChange={closeInstallDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>
                            {!installTask && t('installingService')}
                            {installTask?.status === 'installing' && t('installingService')}
                            {installTask?.status === 'success' && t('installationComplete')}
                            {installTask?.status === 'error' && t('installationFailed')}
                        </DialogTitle>
                        <DialogDescription>
                            {!installTask && `${t('installing')} ${selectedService.name} ${t('from')} ${selectedService.source}`}
                            {installTask?.status === 'installing' && `${t('installing')} ${selectedService.name} ${t('from')} ${selectedService.source}`}
                            {installTask?.status === 'success' && t('theServiceWasInstalledSuccessfully')}
                            {installTask?.status === 'error' && t('thereWasAProblemDuringInstallation')}
                        </DialogDescription>
                    </DialogHeader>

                    <div className="my-4">
                        <div className="bg-muted p-4 rounded-md h-64 overflow-y-auto font-mono text-sm">
                            {(installTask?.logs || []).map((log, index) => (
                                <div key={index} className="pb-1">
                                    <span className="text-primary">{'>'}</span> {log}
                                </div>
                            ))}
                            {(!installTask || installTask.status === 'installing') && (
                                <div className="animate-pulse">
                                    <span className="text-primary">{'>'}</span> _
                                </div>
                            )}
                        </div>
                    </div>

                    <DialogFooter className="flex items-center justify-between">
                        {(!installTask || installTask.status === 'installing') && (
                            <div className="flex items-center text-sm text-muted-foreground">
                                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary mr-2"></div>
                                {t('installing')}...
                            </div>
                        )}
                        {installTask?.status === 'success' && (
                            <div className="flex items-center text-sm text-green-500">
                                <CheckCircle className="h-4 w-4 mr-2" />
                                {t('installationComplete')}
                            </div>
                        )}
                        {installTask?.status === 'error' && (
                            <div className="flex items-center text-sm text-red-500">
                                <XCircle className="h-4 w-4 mr-2" />
                                {t('installationFailed')}: {installTask.error}
                            </div>
                        )}

                        <Button
                            disabled={!installTask || installTask.status === 'installing'}
                            onClick={closeInstallDialog}
                        >
                            {installTask?.status === 'success' ? t('finish') : t('close')}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* 环境变量输入模态框 */}
            <EnvVarInputModal
                open={envModalVisible}
                missingVars={missingVars}
                onSubmit={handleEnvModalSubmit}
                onCancel={handleEnvModalCancel}
            // Optional: Pass service name if your modal supports it for better UX
            // serviceName={selectedService?.name}
            />
        </div>
    );
}
