import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { useToast } from '@/hooks/use-toast';
import { ChevronLeft, Package, Star, Download, ArrowRight, CheckCircle, XCircle, AlertCircle } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';

// 详情页所需的服务类型
type ServiceType = {
    id: string;
    name: string;
    description: string;
    version: string;
    source: 'npm' | 'pypi' | 'recommended';
    author: string;
    downloads: number;
    stars: number;
    readme: string;
    envVars: EnvVarType[];
};

// 环境变量类型
type EnvVarType = {
    name: string;
    description: string;
    isSecret: boolean;
    isRequired: boolean;
    defaultValue?: string;
    value?: string;
};

// 示例数据
const mockServiceDetails: ServiceType = {
    id: '1',
    name: '@modelcontextprotocol/server-airtable',
    description: 'MCP server for Airtable integration. This server allows you to connect your AI models to Airtable databases, making your structured data accessible for context retrieval, querying, and modification.',
    version: '1.0.4',
    source: 'npm',
    author: 'MCP Team',
    downloads: 12540,
    stars: 145,
    readme: `# MCP Airtable Server

This server connects to Airtable and exposes your data via the Model Context Protocol.

## Features

- Connect to multiple Airtable bases
- Query tables with natural language
- Update records through MCP
- Automatic schema detection
- Efficient data retrieval

## Requirements

- Node.js 14+
- Airtable API key
- Airtable base ID

## Quick Start

\`\`\`
npx @modelcontextprotocol/server-airtable
\`\`\`

## Environment Variables

- AIRTABLE_API_KEY - Your Airtable API key
- AIRTABLE_BASE_ID - ID of your Airtable base
- PORT - Optional, defaults to 3000
`,
    envVars: [
        {
            name: 'AIRTABLE_API_KEY',
            description: 'API key for Airtable authentication',
            isSecret: true,
            isRequired: true
        },
        {
            name: 'AIRTABLE_BASE_ID',
            description: 'ID of your Airtable base',
            isSecret: false,
            isRequired: true
        },
        {
            name: 'PORT',
            description: 'Port for the MCP server to listen on',
            isSecret: false,
            isRequired: false,
            defaultValue: '3000'
        }
    ]
};

// 安装状态类型
type InstallStatus = 'idle' | 'installing' | 'success' | 'error';

export function ServiceDetails({ serviceId, onBack }: { serviceId: string, onBack: () => void }) {
    const { toast } = useToast();
    const [service, setService] = useState<ServiceType | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [envVars, setEnvVars] = useState<EnvVarType[]>([]);
    const [showInstallDialog, setShowInstallDialog] = useState(false);
    const [installStatus, setInstallStatus] = useState<InstallStatus>('idle');
    const [installLog, setInstallLog] = useState<string[]>([]);

    // 模拟获取服务详情
    useEffect(() => {
        const fetchServiceDetails = async () => {
            setIsLoading(true);
            // 模拟API请求延迟
            await new Promise(resolve => setTimeout(resolve, 1000));

            // 直接使用模拟数据
            setService(mockServiceDetails);
            setEnvVars(mockServiceDetails.envVars.map(env => ({ ...env })));
            setIsLoading(false);
        };

        fetchServiceDetails();
    }, [serviceId]);

    // 处理环境变量输入变化
    const handleEnvVarChange = (index: number, value: string) => {
        const updatedEnvVars = [...envVars];
        updatedEnvVars[index].value = value;
        setEnvVars(updatedEnvVars);
    };

    // 启动安装流程
    const startInstallation = () => {
        setShowInstallDialog(true);
        setInstallStatus('installing');
        setInstallLog([]);

        // 模拟安装过程
        const installSteps = [
            'Preparing installation environment...',
            `Installing ${service?.name} from ${service?.source}...`,
            'Running package manager...',
            'Creating MCP service record...',
            'Setting environment variables...',
            'Testing connection...',
            'Registering service with MCP system...'
        ];

        // 模拟每个步骤的执行
        let step = 0;
        const intervalId = setInterval(() => {
            if (step < installSteps.length) {
                setInstallLog(prev => [...prev, installSteps[step]]);
                step++;
            } else {
                clearInterval(intervalId);
                setInstallStatus('success');
                setInstallLog(prev => [...prev, 'Installation completed successfully!']);
            }
        }, 800);
    };

    // 关闭安装对话框
    const closeInstallDialog = () => {
        if (installStatus !== 'installing') {
            setShowInstallDialog(false);
            // 如果安装成功，返回上一级
            if (installStatus === 'success') {
                onBack();
                toast({
                    title: "Installation Successful",
                    description: `${service?.name} has been installed and is ready to use.`
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

    // 加载状态
    if (isLoading) {
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
    if (!service) {
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
                    <h1 className="text-3xl font-bold break-words mb-2">{service.name}</h1>
                    <div className="flex flex-wrap items-center gap-4 mt-2 text-sm text-muted-foreground">
                        <div>v{service.version}</div>
                        <div className="flex items-center gap-1">
                            <Download className="h-3.5 w-3.5" />
                            <span>{service.downloads.toLocaleString()}</span>
                        </div>
                        <div className="flex items-center gap-1">
                            <Star className="h-3.5 w-3.5" />
                            <span>{service.stars}</span>
                        </div>
                        <div>
                            <span>By {service.author}</span>
                        </div>
                        <div>
                            <span>Source: {service.source}</span>
                        </div>
                    </div>
                    <p className="mt-4 text-balance">{service.description}</p>
                </div>

                <Button onClick={startInstallation} className="md:self-start flex-shrink-0">
                    Install Service
                </Button>
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
                                {service.readme.split('\n').map((line, index) => {
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
                                {envVars.map((envVar, index) => (
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
                            {installStatus === 'installing' && 'Installing Service...'}
                            {installStatus === 'success' && 'Installation Complete'}
                            {installStatus === 'error' && 'Installation Failed'}
                        </DialogTitle>
                        <DialogDescription>
                            {installStatus === 'installing' && `Installing ${service.name} from ${service.source}`}
                            {installStatus === 'success' && 'The service was installed successfully'}
                            {installStatus === 'error' && 'There was a problem during installation'}
                        </DialogDescription>
                    </DialogHeader>

                    <div className="my-4">
                        <div className="bg-muted p-4 rounded-md h-64 overflow-y-auto font-mono text-sm">
                            {installLog.map((log, index) => (
                                <div key={index} className="pb-1">
                                    <span className="text-primary">{'>'}</span> {log}
                                </div>
                            ))}
                            {installStatus === 'installing' && (
                                <div className="animate-pulse">
                                    <span className="text-primary">{'>'}</span> _
                                </div>
                            )}
                        </div>
                    </div>

                    <DialogFooter className="flex items-center justify-between">
                        {installStatus === 'installing' && (
                            <div className="flex items-center text-sm text-muted-foreground">
                                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary mr-2"></div>
                                Installing...
                            </div>
                        )}
                        {installStatus === 'success' && (
                            <div className="flex items-center text-sm text-green-500">
                                <CheckCircle className="h-4 w-4 mr-2" />
                                Installation complete
                            </div>
                        )}
                        {installStatus === 'error' && (
                            <div className="flex items-center text-sm text-red-500">
                                <XCircle className="h-4 w-4 mr-2" />
                                Installation failed
                            </div>
                        )}

                        <Button
                            disabled={installStatus === 'installing'}
                            onClick={closeInstallDialog}
                        >
                            {installStatus === 'success' ? 'Finish' : 'Close'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
} 