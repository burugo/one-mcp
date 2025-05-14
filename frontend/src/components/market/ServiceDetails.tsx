import { useState } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { useToast } from '@/hooks/use-toast';
import { ChevronLeft, Package, Star, Download, CheckCircle, XCircle, AlertCircle } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { useMarketStore } from '@/store/marketStore';



export function ServiceDetails({ onBack }: { onBack: () => void }) {
    const { toast } = useToast();
    const {
        selectedService,
        isLoadingDetails,
        installTasks,
        updateEnvVar,
        installService,
        uninstallService
    } = useMarketStore();

    // 显示安装对话框状态
    const [showInstallDialog, setShowInstallDialog] = useState(false);

    // 获取当前服务的安装任务（如果有）
    const installTask = selectedService ?
        installTasks[selectedService.id] : undefined;

    // 处理环境变量输入变化
    const handleEnvVarChange = (index: number, value: string) => {
        if (selectedService) {
            updateEnvVar(
                selectedService.id,
                selectedService.envVars[index].name,
                value
            );
        }
    };

    // 启动安装流程
    const startInstallation = () => {
        if (!selectedService) return;

        setShowInstallDialog(true);

        // 准备环境变量
        const envVarsObj: { [key: string]: string } = {};
        selectedService.envVars.forEach(env => {
            if (env.value) {
                envVarsObj[env.name] = env.value;
            }
        });

        // 调用安装方法
        installService(selectedService.id, envVarsObj);
    };

    // 关闭安装对话框
    const closeInstallDialog = () => {
        if (installTask?.status !== 'installing') {
            setShowInstallDialog(false);

            // 如果安装成功，返回上一级
            if (installTask?.status === 'success') {
                onBack();
                toast({
                    title: "Installation Successful",
                    description: `${selectedService?.name} has been installed and is ready to use.`
                });
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
    const handleUninstall = () => {
        if (!selectedService) return;

        // 显示确认对话框
        if (window.confirm(`Are you sure you want to uninstall ${selectedService.name}?`)) {
            uninstallService(selectedService.id);
            onBack(); // 返回市场页面
            toast({
                title: "Service Uninstalled",
                description: `${selectedService.name} has been uninstalled.`
            });
        }
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
                    <h1 className="text-3xl font-bold break-words mb-2">{selectedService.name}</h1>
                    <div className="flex flex-wrap items-center gap-4 mt-2 text-sm text-muted-foreground">
                        <div>v{selectedService.version}</div>
                        <div className="flex items-center gap-1">
                            <Download className="h-3.5 w-3.5" />
                            <span>{selectedService.downloads.toLocaleString()}</span>
                        </div>
                        <div className="flex items-center gap-1">
                            <Star className="h-3.5 w-3.5" />
                            <span>{selectedService.stars}</span>
                        </div>
                        <div>
                            <span>By {selectedService.author}</span>
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
                    <Button onClick={startInstallation} className="md:self-start flex-shrink-0">
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
                                {selectedService.readme.split('\n').map((line, index) => {
                                    if (line.startsWith('# ')) {
                                        return <h1 key={index} className="text-2xl font-bold mt-4 mb-2">{line.substring(2)}</h1>;
                                    } else if (line.startsWith('## ')) {
                                        return <h2 key={index} className="text-xl font-semibold mt-4 mb-2">{line.substring(3)}</h2>;
                                    } else if (line.startsWith('```')) {
                                        return (
                                            <pre key={index} className="bg-muted p-4 rounded-md my-4 overflow-x-auto">
                                                <code>{line.substring(3)}</code>
                                            </pre>
                                        );
                                    } else if (line.startsWith('- ')) {
                                        return <li key={index} className="ml-4">{line.substring(2)}</li>;
                                    } else if (line === '') {
                                        return <br key={index} />;
                                    } else {
                                        return <p key={index} className="my-2">{line}</p>;
                                    }
                                })}
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
                                {selectedService.envVars.map((envVar, index) => (
                                    <div key={index} className="grid grid-cols-1 md:grid-cols-4 gap-4 items-center">
                                        <div className="md:col-span-1">
                                            <label htmlFor={`env-${index}`} className="text-sm font-medium flex items-center gap-1">
                                                {envVar.name}
                                                {envVar.isRequired && <span className="text-red-500">*</span>}
                                            </label>
                                        </div>
                                        <div className="md:col-span-3">
                                            <Input
                                                id={`env-${index}`}
                                                type={envVar.isSecret ? "password" : "text"}
                                                placeholder={envVar.defaultValue ? `Default: ${envVar.defaultValue}` : ""}
                                                value={envVar.value || ""}
                                                onChange={(e) => handleEnvVarChange(index, e.target.value)}
                                                className="w-full"
                                            />
                                            <p className="text-xs text-muted-foreground mt-1">{envVar.description}</p>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </CardContent>
                        <CardFooter>
                            <Button onClick={startInstallation} className="ml-auto">
                                Install with Configuration
                            </Button>
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
                            {installTask?.logs.map((log, index) => (
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
        </div>
    );
} 