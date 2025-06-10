import React, { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Check, Copy } from 'lucide-react';
import { useServerAddress } from '@/hooks/useServerAddress';
import { useAuth } from '@/contexts/AuthContext';
import { useToast } from '@/hooks/use-toast';

interface ServiceConfigModalProps {
    open: boolean;
    service: any; // 具体类型可根据实际服务对象定义
    onClose: () => void;
    onSaveVar: (varName: string, value: string) => Promise<void>;
}

function getEnvVars(service: any): Record<string, string> {
    if (!service) return {};
    if (service.env_vars && typeof service.env_vars === 'object') return service.env_vars;
    return {};
}

const ServiceConfigModal: React.FC<ServiceConfigModalProps> = ({ open, service, onClose, onSaveVar }) => {
    const [envValues, setEnvValues] = useState<Record<string, string>>(getEnvVars(service));
    const [saving, setSaving] = useState<string | null>(null);
    const [copied, setCopied] = useState<{ [k: string]: boolean }>({});
    const [error, setError] = useState<string | null>(null);
    const [userToken, setUserToken] = useState<string>('');
    const serverAddress = useServerAddress();
    const { currentUser, updateUserInfo } = useAuth();
    const { toast } = useToast();

    React.useEffect(() => {
        setEnvValues(getEnvVars(service));
    }, [service]);

    // 获取用户token
    React.useEffect(() => {
        const fetchUserToken = async () => {
            try {
                // 首先检查currentUser中是否已有token
                if (currentUser?.token) {
                    setUserToken(currentUser.token);
                    return;
                }

                // 如果没有，从API获取
                const response = await fetch('/api/user/self', {
                    headers: {
                        'Authorization': `Bearer ${localStorage.getItem('token')}`,
                        'Content-Type': 'application/json'
                    }
                });

                if (response.ok) {
                    const data = await response.json();
                    if (data.success && data.data?.token) {
                        setUserToken(data.data.token);
                        // 更新AuthContext中的用户信息
                        if (currentUser) {
                            updateUserInfo({
                                ...currentUser,
                                token: data.data.token
                            });
                        }
                    }
                }
            } catch (error) {
                console.error('Failed to fetch user token:', error);
            }
        };

        if (open && currentUser) {
            fetchUserToken();
        }
    }, [open, currentUser, updateUserInfo]);

    const handleChange = (name: string, value: string) => {
        setEnvValues((prev) => ({ ...prev, [name]: value }));
    };

    const handleSave = async (varName: string) => {
        setSaving(varName);
        setError(null);
        try {
            await onSaveVar(varName, envValues[varName]);
        } catch (e: any) {
            setError(e.message || '保存失败');
        }
        setSaving(null);
    };

    // 检查用户是否是管理员(role >= 10)
    const isAdmin = currentUser?.role && currentUser.role >= 10;

    // 生成 endpoint
    const sseEndpoint = serverAddress ? `${serverAddress}/proxy/${service?.name || ''}/sse${userToken ? `?key=${userToken}` : ''}` : '';
    const httpEndpoint = serverAddress ? `${serverAddress}/proxy/${service?.name || ''}/mcp${userToken ? `?key=${userToken}` : ''}` : '';

    // 生成 SSE JSON 配置
    const generateSSEJSONConfig = () => {
        const serviceName = service?.name || 'unknown-service';
        const config = {
            mcpServers: {
                [serviceName]: {
                    url: sseEndpoint
                }
            }
        };
        return JSON.stringify(config, null, 2);
    };

    // 生成 HTTP JSON 配置
    const generateHTTPJSONConfig = () => {
        const serviceName = service?.name || 'unknown-service';
        const config = {
            mcpServers: {
                [serviceName]: {
                    url: httpEndpoint
                }
            }
        };
        return JSON.stringify(config, null, 2);
    };

    const handleCopySSE = async () => {
        const jsonConfig = generateSSEJSONConfig();
        try {
            await navigator.clipboard.writeText(jsonConfig);
            setCopied((prev) => ({ ...prev, 'sse': true }));
            setTimeout(() => setCopied((prev) => ({ ...prev, 'sse': false })), 1200);
            toast({
                title: "SSE配置已复制",
                description: "SSE端点的MCP服务器配置已复制到剪贴板"
            });
        } catch {
            toast({
                variant: "destructive",
                title: "复制失败",
                description: "无法访问剪贴板"
            });
        }
    };

    const handleCopyHTTP = async () => {
        const jsonConfig = generateHTTPJSONConfig();
        try {
            await navigator.clipboard.writeText(jsonConfig);
            setCopied((prev) => ({ ...prev, 'http': true }));
            setTimeout(() => setCopied((prev) => ({ ...prev, 'http': false })), 1200);
            toast({
                title: "HTTP配置已复制",
                description: "HTTP端点的MCP服务器配置已复制到剪贴板"
            });
        } catch {
            toast({
                variant: "destructive",
                title: "复制失败",
                description: "无法访问剪贴板"
            });
        }
    };

    const handleUpdateRPDLimit = async (newLimit: number) => {
        if (!service?.id) return;

        try {
            const response = await fetch(`/api/mcp_services/${service.id}`, {
                method: 'PUT',
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('token')}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    rpd_limit: newLimit
                })
            });

            if (!response.ok) {
                throw new Error('更新RPD限制失败');
            }

            // 更新本地service对象
            service.rpd_limit = newLimit;

            // 重新计算剩余请求数
            if (newLimit > 0 && service.user_daily_request_count !== undefined) {
                service.remaining_requests = newLimit - (service.user_daily_request_count || 0);
            } else {
                service.remaining_requests = -1; // 无限制
            }

            toast({
                title: "更新成功",
                description: `每日请求限制已更新为 ${newLimit === 0 ? '无限制' : `${newLimit}次/天`}`
            });
        } catch (error) {
            toast({
                variant: "destructive",
                title: "更新失败",
                description: error instanceof Error ? error.message : "更新RPD限制时发生错误"
            });
        }
    };

    return (
        <Dialog open={open} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader className="mb-4">
                    <DialogTitle>Service Configuration</DialogTitle>
                    <DialogDescription>
                        Adjust the settings for this service. Click save when you're done.
                    </DialogDescription>
                </DialogHeader>

                {/* 环境变量配置部分 - 只有管理员可以看到 */}
                {isAdmin && (
                    <div className="space-y-4 mt-2">
                        <div className="text-sm font-medium text-foreground mb-2">Environment Variables</div>
                        {Object.keys(envValues).length === 0 && (
                            <div className="text-muted-foreground text-sm">No environment variables for this service.</div>
                        )}
                        {Object.keys(envValues).map((varName) => (
                            <div key={varName} className="mb-4">
                                <label
                                    className="block text-sm font-medium mb-1 break-all"
                                    title={varName}
                                >
                                    {varName}
                                </label>
                                <div className="flex gap-2 items-center">
                                    <Input
                                        type="text"
                                        value={envValues[varName] || ''}
                                        onChange={(e) => handleChange(varName, e.target.value)}
                                        className="flex-1 min-w-0"
                                    />
                                    <Button
                                        size="sm"
                                        variant="secondary"
                                        onClick={() => handleSave(varName)}
                                        disabled={saving === varName}
                                    >
                                        {saving === varName ? 'Saving...' : 'Save'}
                                    </Button>
                                </div>
                            </div>
                        ))}
                        {error && <div className="text-red-500 text-sm mt-2">{error}</div>}
                        {Object.keys(envValues).length > 0 && <div className="my-4 border-t border-border"></div>}
                    </div>
                )}

                {/* 端点地址部分 - 所有用户都可以看到 */}
                <div className="space-y-3">
                    <div className="text-sm font-medium text-foreground mb-2">Service Endpoints</div>
                    <div className="flex items-center gap-2">
                        <span className="w-28 text-sm font-medium">SSE Endpoint</span>
                        <Input value={sseEndpoint} readOnly className="flex-1" />
                        <Button
                            size="icon"
                            variant="ghost"
                            onClick={handleCopySSE}
                            disabled={!sseEndpoint}
                            title="复制SSE配置"
                        >
                            {copied['sse'] ? <Check className="text-green-500 w-5 h-5" /> : <Copy className="w-5 h-5" />}
                        </Button>
                    </div>
                    <div className="flex items-center gap-2">
                        <span className="w-28 text-sm font-medium">HTTP Endpoint</span>
                        <Input value={httpEndpoint} readOnly className="flex-1" />
                        <Button
                            size="icon"
                            variant="ghost"
                            onClick={handleCopyHTTP}
                            disabled={!httpEndpoint}
                            title="复制HTTP配置"
                        >
                            {copied['http'] ? <Check className="text-green-500 w-5 h-5" /> : <Copy className="w-5 h-5" />}
                        </Button>
                    </div>
                </div>

                {/* 每日请求限制 (RPD) 配置 */}
                <div className="space-y-3 mt-4 pt-3 border-t border-border">
                    <div className="text-sm font-medium text-foreground">每日请求次数限制 (RPD)</div>
                    <div className="flex items-center gap-2">
                        <span className="text-sm text-muted-foreground">当前限制:</span>
                        {isAdmin ? (
                            <Input
                                type="number"
                                min="0"
                                value={service?.rpd_limit || 0}
                                onChange={(e) => handleUpdateRPDLimit(parseInt(e.target.value) || 0)}
                                placeholder="0 表示不限制"
                                className="w-32"
                            />
                        ) : (
                            <span className="font-medium">
                                {service?.rpd_limit === 0 || !service?.rpd_limit ? 'no limit' : service?.rpd_limit}
                            </span>
                        )}
                        <span className="text-sm text-muted-foreground">
                            {service?.rpd_limit === 0 || !service?.rpd_limit ? '(无限制)' : '次/天'}
                        </span>
                    </div>
                </div>

                <DialogFooter className="mt-4">
                    <Button variant="outline" onClick={onClose} type="button">Close</Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
};

export default ServiceConfigModal; 