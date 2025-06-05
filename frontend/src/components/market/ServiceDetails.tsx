import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { useToast } from '@/hooks/use-toast';
import { ChevronLeft, Package, Star, Download, CheckCircle, XCircle, AlertCircle } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { useMarketStore } from '@/store/marketStore';
import EnvVarInputModal from './EnvVarInputModal';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import api from '@/utils/api'; // Import api utility for saving

export function ServiceDetails({ onBack }: { onBack: () => void }) {
    const { toast } = useToast();
    const {
        selectedService,
        isLoadingDetails,
        installTasks,
        updateEnvVar,
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

    // 处理环境变量输入变化
    const handleEnvVarChange = (varName: string, value: string) => {
        if (selectedService) {
            updateEnvVar(
                selectedService.id,
                varName,
                value
            );
        }
    };

    // Helper function to get placeholder for env var
    const getEnvVarPlaceholder = (envVar: any): string => {
        // Try to get placeholder from mcp_config first
        if (selectedService?.mcpConfig?.mcpServers) {
            const servers = selectedService.mcpConfig.mcpServers;
            for (const serverKey in servers) {
                const server = servers[serverKey];
                if (server.env && server.env[envVar.name]) {
                    const configValue = server.env[envVar.name];
                    // If it looks like a placeholder (contains "your-" or similar patterns)
                    if (typeof configValue === 'string' &&
                        (configValue.toLowerCase().includes('your-') ||
                            configValue.toLowerCase().includes('enter-') ||
                            configValue.toLowerCase().includes('api-key') ||
                            configValue.toLowerCase().includes('token'))) {
                        return configValue;
                    }
                }
            }
        }

        // Fallback to default value or generated placeholder
        if (envVar.defaultValue && envVar.defaultValue.trim() !== '') {
            return envVar.defaultValue;
        }

        // Generate placeholder based on variable name and type
        if (envVar.isSecret) {
            if (envVar.name.toLowerCase().includes('token')) {
                return 'your-api-token';
            } else if (envVar.name.toLowerCase().includes('key')) {
                return 'your-api-key';
            } else {
                return 'Enter secret value';
            }
        }

        return 'Enter value';
    };

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
        } else { // If not from modal, collect from Configuration tab
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
        //setCurrentEnvVars({}); // Keep current env vars from config tab if user cancels
        setPendingServiceId(null);
    };

    // 关闭安装对话框
    const closeInstallDialog = async () => {
        if (installTask?.status !== 'installing') {
            setShowInstallDialog(false);

            // 如果安装成功，刷新当前页面状态而不是返回上一级
            if (installTask?.status === 'success') {
                toast({
                    title: "Installation Successful",
                    description: `${selectedService?.name} has been installed and is ready to use.`
                });

                // 刷新当前服务详情以更新安装状态
                if (selectedService) {
                    await fetchServiceDetails(selectedService.id, selectedService.name, selectedService.source);
                }
            }
        } else {
            toast({
                title: "Installation in Progress",
                description: "Please wait for the installation to complete.",
                variant: "destructive"
            });
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

            } catch (error: any) {
                toast({
                    title: "Uninstall Failed",
                    description: error.message || "Failed to uninstall service.",
                    variant: "destructive"
                });
            }
        }
    };

    const handleSaveConfiguration = async () => {
        if (!selectedService || !selectedService.isInstalled || typeof selectedService.installed_service_id !== 'number') {
            toast({
                title: "Cannot Save Configuration",
                description: "Service is not installed or numeric service ID is missing.",
                variant: "destructive"
            });
            return;
        }

        toast({ title: "Saving Configuration...", description: "Please wait." });
        // let allSucceeded = true;
        try {
            for (const envVar of (selectedService.envVars || [])) {
                // Only save if a value is present, or handle as needed
                // Assuming 'value' contains the current value from the input field
                // The API structure for saving might be one var at a time or bulk
                // Using the single var save endpoint as seen in ServicesPage.tsx
                await api.patch('/mcp_market/env_var', {
                    service_id: selectedService.installed_service_id,
                    var_name: envVar.name,
                    var_value: envVar.value || '' // Send empty string if undefined/null
                });
            }
            toast({ title: "Configuration Saved", description: "Environment variables have been updated." });
        } catch (error: any) {
            // allSucceeded = false;
            console.error("Failed to save environment variable:", error);
            toast({
                title: "Save Failed",
                description: error.message || "Could not save one or more environment variables.",
                variant: "destructive"
            });
        }
        // Optionally, re-fetch service details or update local state if necessary
        // if (allSucceeded && selectedService) {
        //     get().fetchServiceDetails(selectedService.id, selectedService.name, selectedService.source);
        // }
    };

    // 加载状态
    if (isLoadingDetails) {
        return (
            <div className="flex-1 p-6 flex justify-center items-center">
                <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto"></div>
                    <p className="mt-4 text-muted-foreground">Loading service details...</p>
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
                    Back to Marketplace
                </Button>
                <div className="text-center py-12">
                    <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                    <h3 className="text-xl font-semibold mb-2">Service Not Found</h3>
                    <p className="text-muted-foreground">The requested service could not be found.</p>
                </div>
            </div>
        );
    }

    return (
        <div className="flex-1 space-y-6">
            <Button variant="ghost" onClick={onBack} className="mb-2">
                <ChevronLeft className="mr-2 h-4 w-4" />
                Back to Marketplace
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
                            <span>By {typeof selectedService.author === 'string' ? selectedService.author : selectedService.author?.name || 'Unknown Author'}</span>
                        </div>
                        <div>
                            <span>Source: {selectedService.source}</span>
                        </div>
                    </div>
                    <p className="mt-4 text-balance">{selectedService.description}</p>
                </div>

                {selectedService.isInstalled ? (
                    <Button onClick={handleUninstall} variant="destructive" className="md:self-start flex-shrink-0">
                        Uninstall Service
                    </Button>
                ) : (
                    <Button onClick={() => startInstallation()} className="md:self-start flex-shrink-0">
                        Install Service
                    </Button>
                )}
            </div>

            {/* 详情选项卡 */}
            <Tabs defaultValue="readme" className="mt-8">
                <TabsList>
                    <TabsTrigger value="readme">README</TabsTrigger>
                    <TabsTrigger value="configuration">Configuration</TabsTrigger>
                </TabsList>

                <TabsContent value="readme" className="mt-4">
                    <Card>
                        <CardContent className="pt-6">
                            <div className="prose dark:prose-invert max-w-none">
                                <ReactMarkdown remarkPlugins={[remarkGfm]}>
                                    {selectedService.readme}
                                </ReactMarkdown>
                            </div>
                        </CardContent>
                    </Card>
                </TabsContent>

                <TabsContent value="configuration" className="mt-4">
                    <Card>
                        <CardHeader>
                            <CardTitle>Environment Variables</CardTitle>
                            <CardDescription>
                                Configure the required environment variables for this service.
                            </CardDescription>
                        </CardHeader>
                        <CardContent>
                            <div className="space-y-4">
                                {(selectedService.envVars || []).map((envVar, index) => (
                                    <div key={index} className="grid grid-cols-1 md:grid-cols-4 gap-4 items-center">
                                        <div className="md:col-span-1">
                                            <label htmlFor={`env-${envVar.name}`} className="text-sm font-medium flex items-center gap-1">
                                                {envVar.name}
                                                {!envVar.optional && <span className="text-red-500">*</span>}
                                            </label>
                                        </div>
                                        <div className="md:col-span-3">
                                            <Input
                                                id={`env-${envVar.name}`}
                                                type="text"
                                                placeholder={getEnvVarPlaceholder(envVar)}
                                                value={envVar.value || ''}
                                                onChange={(e) => handleEnvVarChange(envVar.name, e.target.value)}
                                                className="w-full"
                                            />
                                            <p className="text-xs text-muted-foreground mt-1">{envVar.description}</p>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </CardContent>
                        <CardFooter>
                            {selectedService && selectedService.isInstalled ? (
                                <Button onClick={handleSaveConfiguration} className="ml-auto">
                                    Save Configuration
                                </Button>
                            ) : (
                                <Button onClick={() => startInstallation()} className="ml-auto">
                                    Install with Configuration
                                </Button>
                            )}
                        </CardFooter>
                    </Card>
                </TabsContent>
            </Tabs>

            {/* 安装进度对话框 */}
            <Dialog open={showInstallDialog} onOpenChange={closeInstallDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>
                            {!installTask && 'Installing Service...'}
                            {installTask?.status === 'installing' && 'Installing Service...'}
                            {installTask?.status === 'success' && 'Installation Complete'}
                            {installTask?.status === 'error' && 'Installation Failed'}
                        </DialogTitle>
                        <DialogDescription>
                            {!installTask && `Installing ${selectedService.name} from ${selectedService.source}`}
                            {installTask?.status === 'installing' && `Installing ${selectedService.name} from ${selectedService.source}`}
                            {installTask?.status === 'success' && 'The service was installed successfully'}
                            {installTask?.status === 'error' && 'There was a problem during installation'}
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
                                Installing...
                            </div>
                        )}
                        {installTask?.status === 'success' && (
                            <div className="flex items-center text-sm text-green-500">
                                <CheckCircle className="h-4 w-4 mr-2" />
                                Installation complete
                            </div>
                        )}
                        {installTask?.status === 'error' && (
                            <div className="flex items-center text-sm text-red-500">
                                <XCircle className="h-4 w-4 mr-2" />
                                Installation failed: {installTask.error}
                            </div>
                        )}

                        <Button
                            disabled={!installTask || installTask.status === 'installing'}
                            onClick={closeInstallDialog}
                        >
                            {installTask?.status === 'success' ? 'Finish' : 'Close'}
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