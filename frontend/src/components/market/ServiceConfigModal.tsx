import React, { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Check, Copy, AlertCircle } from 'lucide-react';
import { useServerAddress } from '@/hooks/useServerAddress';
import { useAuth } from '@/contexts/AuthContext';
import { useToast } from '@/hooks/use-toast';
import { useTranslation } from 'react-i18next';
import { copyToClipboard, getClipboardErrorMessage, isClipboardSupported } from '@/utils/clipboard';
import { Textarea } from '@/components/ui/textarea';
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

// Define a more specific type for the service object
interface Service {
    id: string | number; // 兼容 ServiceType 的 string 类型 id
    name: string;
    display_name?: string; // 显示名称
    env_vars?: Record<string, string>;
    rpd_limit?: number; // 改为可选
    user_daily_request_count?: number;
    remaining_requests?: number;
    // Add other properties from ServiceType to ensure compatibility
    version?: string;
    description?: string;
    source?: string;
    isInstalled?: boolean;
    installed_service_id?: number;
    // Add other properties from the service object as needed
}

interface ServiceConfigModalProps {
    open: boolean;
    service: Service | null; // Use the specific type
    onClose: () => void;
}



const ServiceConfigModal: React.FC<ServiceConfigModalProps> = ({ open, service, onClose }) => {
    const { t } = useTranslation();
    const [copied, setCopied] = useState<{ [k: string]: boolean }>({});
    const [userToken, setUserToken] = useState<string>('');
    const [showManualCopy, setShowManualCopy] = useState<{ [k: string]: boolean }>({});
    const serverAddress = useServerAddress();
    const { currentUser, updateUserInfo } = useAuth();
    const { toast } = useToast();
    const [selectedEndpointType, setSelectedEndpointType] = useState<'sse' | 'streamableHttp'>('streamableHttp');



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



    // 检查用户是否是管理员(role >= 10)
    const isAdmin = currentUser?.role && currentUser.role >= 10;

    // 生成 endpoint
    const sseEndpoint = serverAddress ? `${serverAddress}/proxy/${service?.name || ''}/sse` : '';
    const httpEndpoint = serverAddress ? `${serverAddress}/proxy/${service?.name || ''}/mcp${userToken ? `?key=${userToken}` : ''}` : '';

    // 生成 SSE JSON 配置
    const generateSSEJSONConfig = () => {
        const serviceName = service?.name || 'unknown-service';
        const serverConfig: {
            type: 'sse';
            url: string;
            header?: { Authorization: string };
        } = {
            type: 'sse',
            url: sseEndpoint
        };

        // 如果有用户token，添加Authorization header
        if (userToken) {
            serverConfig.header = {
                "Authorization": `Bearer ${userToken}`
            };
        }

        const config = {
            mcpServers: {
                [serviceName]: serverConfig
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
                    type: 'streamableHttp',
                    url: httpEndpoint
                }
            }
        };
        return JSON.stringify(config, null, 2);
    };

    // copy endpoint url
    const handleCopyEndpoint = async () => {
        const urlToCopy = selectedEndpointType === 'sse' ? sseEndpoint : httpEndpoint;
        const result = await copyToClipboard(urlToCopy);

        if (result.success) {
            setCopied((prev) => ({ ...prev, 'endpoint': true }));
            setTimeout(() => setCopied((prev) => ({ ...prev, 'endpoint': false })), 1200);
            toast({
                title: t('serviceConfigModal.messages.endpointUrlCopied'),
                description: t('serviceConfigModal.messages.endpointUrlCopiedDesc')
            });
        } else {
            const errorMessageKey = getClipboardErrorMessage(result.error);
            toast({
                variant: "destructive",
                title: t('serviceConfigModal.messages.copyFailed'),
                description: t(errorMessageKey)
            });
        }
    };

    // copy json config
    const handleCopyJSON = async () => {
        const jsonConfig = selectedEndpointType === 'sse' ? generateSSEJSONConfig() : generateHTTPJSONConfig();
        const result = await copyToClipboard(jsonConfig);

        if (result.success) {
            setCopied((prev) => ({ ...prev, 'json': true }));
            setTimeout(() => setCopied((prev) => ({ ...prev, 'json': false })), 1200);
            toast({
                title: t('serviceConfigModal.messages.jsonConfigCopied'),
                description: t('serviceConfigModal.messages.jsonConfigCopiedDesc')
            });
        } else {
            // 
            setShowManualCopy((prev) => ({ ...prev, [selectedEndpointType]: true }));
            const errorMessageKey = getClipboardErrorMessage(result.error);
            toast({
                variant: "destructive",
                title: t('serviceConfigModal.messages.copyFailed'),
                description: t(errorMessageKey)
            });
        }
    };



    const handleCopyHeaderText = async () => {
        if (!userToken) return;
        const headerText = `Authorization=Bearer ${userToken}`;
        const result = await copyToClipboard(headerText);

        if (result.success) {
            setCopied((prev) => ({ ...prev, 'headerText': true }));
            setTimeout(() => setCopied((prev) => ({ ...prev, 'headerText': false })), 1200);
            toast({
                title: t('serviceConfigModal.messages.headerCopied', 'Header copied to clipboard'), // Placeholder for new translation
            });
        } else {
            const errorMessageKey = getClipboardErrorMessage(result.error);
            toast({
                variant: "destructive",
                title: t('serviceConfigModal.messages.copyFailed'),
                description: t(errorMessageKey)
            });
            // Optionally, implement manual copy for header text if needed, similar to JSON sections
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
                throw new Error(t('serviceConfigModal.messages.updateFailed'));
            }

            // 更新本地service对象
            service.rpd_limit = newLimit;

            // 重新计算剩余请求数
            if (newLimit > 0 && service.user_daily_request_count !== undefined) {
                service.remaining_requests = newLimit - (service.user_daily_request_count || 0);
            } else {
                service.remaining_requests = -1; // 无限制
            }

            const limitText = newLimit === 0 ? t('serviceConfigModal.messages.unlimitedValue') : `${newLimit}${t('serviceConfigModal.sections.requestsPerDay')}`;
            toast({
                title: t('serviceConfigModal.messages.updateSuccess'),
                description: t('serviceConfigModal.messages.rpdLimitUpdated', { limit: limitText })
            });
        } catch (error) {
            toast({
                variant: "destructive",
                title: t('serviceConfigModal.messages.updateFailed'),
                description: error instanceof Error ? error.message : t('serviceConfigModal.messages.rpdUpdateError')
            });
        }
    };

    return (
        <Dialog open={open} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader className="mb-4">
                    <DialogTitle>
                        {service?.display_name || service?.name || t('serviceConfigModal.title')}
                    </DialogTitle>
                    <DialogDescription>
                        {t('serviceConfigModal.description')}
                    </DialogDescription>
                </DialogHeader>



                {/* 端点地址部分 - 所有用户都可以看到 */}
                <div className="space-y-3">
                    <div className="text-sm font-medium text-foreground mb-2">{t('serviceConfigModal.sections.serviceEndpoints')}</div>

                    {/* 安全上下文警告 */}
                    {!isClipboardSupported() && (
                        <div className="flex items-start gap-2 p-3 bg-amber-50 border border-amber-200 rounded-md">
                            <AlertCircle className="w-4 h-4 text-amber-600 mt-0.5 flex-shrink-0" />
                            <div className="text-sm text-amber-800">
                                <p className="font-medium">{t('serviceConfigModal.messages.clipboardNotSupported')}</p>
                                <p className="mt-1">{t('serviceConfigModal.messages.manualCopyHint')}</p>
                            </div>
                        </div>
                    )}

                    {/* Service Type Dropdown */}
                    <div className="space-y-2">
                        <Label htmlFor="endpoint-type" className="text-xs text-muted-foreground">{t('customServiceModal.form.serviceType', 'Service Type')}</Label>
                        <Select
                            value={selectedEndpointType}
                            onValueChange={(value: 'sse' | 'streamableHttp') => setSelectedEndpointType(value)}
                        >
                            <SelectTrigger id="endpoint-type" className="w-full">
                                <SelectValue placeholder="Select endpoint type" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="streamableHttp">{t('customServiceModal.serviceTypes.streamableHttp', 'Streamable HTTP')}</SelectItem>
                                <SelectItem value="sse">{t('customServiceModal.serviceTypes.sse', 'Server-Sent Events (SSE)')}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>

                    {/* Label for Copy Config */}
                    <div className="mt-3">
                        <Label htmlFor="endpoint-url-input" className="text-xs text-muted-foreground">
                            {t('serviceConfigModal.actions.copyConfigLabel', 'Copy Endpoint Config')}
                        </Label>
                    </div>

                    {/* URL display and copy button */}
                    <div className="flex items-center gap-2 mt-1">
                        <Input
                            id="endpoint-url-input"
                            value={selectedEndpointType === 'sse' ? sseEndpoint : httpEndpoint}
                            readOnly
                            className="flex-1"
                            placeholder={t('serviceConfigModal.sections.urlPlaceholder', 'Endpoint URL will appear here')}
                        />
                        <Button
                            size="icon"
                            variant="ghost"
                            onClick={handleCopyEndpoint}
                            disabled={!(selectedEndpointType === 'sse' ? sseEndpoint : httpEndpoint)}
                            title={t('serviceConfigModal.actions.copyEndpointUrl')}
                        >
                            {copied['endpoint'] ? <Check className="text-green-500 w-5 h-5" /> : <Copy className="w-5 h-5" />}
                        </Button>
                    </div>

                    {/* Conditional SSE Header display and its copy button */}
                    {selectedEndpointType === 'sse' && userToken && (
                        <div className="mt-2 flex items-center gap-2">
                            <Input
                                value={`Authorization=Bearer ${userToken}`}
                                readOnly
                                className="flex-1 text-xs font-mono"
                            />
                            <Button
                                size="icon"
                                variant="ghost"
                                onClick={handleCopyHeaderText}
                                title={t('serviceConfigModal.actions.copyHeaderText', 'Copy Header Text')}
                            >
                                {copied['headerText'] ? <Check className="text-green-500 w-5 h-5" /> : <Copy className="w-5 h-5" />}
                            </Button>
                        </div>
                    )}

                    {/* Conditional Manual Copy JSON Area */}
                    {selectedEndpointType === 'sse' && showManualCopy['sse'] && (
                        <div className="space-y-2 mt-2"> {/* Added mt-2 for spacing */}
                            <div className="text-sm font-medium text-foreground">{t('serviceConfigModal.sections.sseConfigJson')}</div>
                            <Textarea
                                value={generateSSEJSONConfig()}
                                readOnly
                                className="font-mono text-xs"
                                rows={8}
                                onClick={(e) => e.currentTarget.select()}
                            />
                            <p className="text-xs text-muted-foreground">{t('serviceConfigModal.messages.selectAllHint')}</p>
                        </div>
                    )}
                    {selectedEndpointType === 'streamableHttp' && showManualCopy['http'] && (
                        <div className="space-y-2 mt-2"> {/* Added mt-2 for spacing */}
                            <div className="text-sm font-medium text-foreground">{t('serviceConfigModal.sections.httpConfigJson')}</div>
                            <Textarea
                                value={generateHTTPJSONConfig()}
                                readOnly
                                className="font-mono text-xs"
                                rows={8}
                                onClick={(e) => e.currentTarget.select()}
                            />
                            <p className="text-xs text-muted-foreground">{t('serviceConfigModal.messages.selectAllHint')}</p>
                        </div>
                    )}
                </div>

                {/* 每日请求限制 (RPD) 配置 */}
                <div className="space-y-3 mt-4 pt-3 border-t border-border">
                    <div className="text-sm font-medium text-foreground">{t('serviceConfigModal.sections.rpdLimit')}</div>
                    <div className="flex items-center gap-2">
                        <span className="text-sm text-muted-foreground">{t('serviceConfigModal.sections.currentLimit')}</span>
                        {isAdmin ? (
                            <Input
                                type="number"
                                min="0"
                                value={service?.rpd_limit || 0}
                                onChange={(e) => handleUpdateRPDLimit(parseInt(e.target.value) || 0)}
                                placeholder={t('serviceConfigModal.actions.limitPlaceholder')}
                                className="w-32"
                            />
                        ) : (
                            <span className="font-medium">
                                {service?.rpd_limit === 0 || !service?.rpd_limit ? t('serviceConfigModal.messages.unlimitedValue') : service?.rpd_limit}
                            </span>
                        )}
                        <span className="text-sm text-muted-foreground">
                            {service?.rpd_limit === 0 || !service?.rpd_limit ? t('serviceConfigModal.sections.unlimited') : t('serviceConfigModal.sections.requestsPerDay')}
                        </span>
                    </div>
                </div>

                <DialogFooter className="mt-4 flex justify-between">
                    <Button variant="outline" onClick={onClose} type="button">{t('serviceConfigModal.actions.close')}</Button>
                    <Button
                        variant="default"
                        onClick={handleCopyJSON}
                        disabled={!(selectedEndpointType === 'sse' ? sseEndpoint : httpEndpoint)}
                        type="button"
                        title={t('serviceConfigModal.actions.copyJsonTooltip')}
                    >
                        {t('serviceConfigModal.actions.copyJson')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
};

export default ServiceConfigModal; 