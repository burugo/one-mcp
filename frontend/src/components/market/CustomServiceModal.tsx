import React, { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { CheckCircle, Loader2 } from 'lucide-react';

interface CustomServiceModalProps {
    open: boolean;
    onClose: () => void;
    onCreateService: (serviceData: CustomServiceData) => Promise<void>;
}

export interface CustomServiceData {
    name: string;
    type: 'stdio' | 'sse' | 'streamableHttp';
    command?: string;
    arguments?: string;
    environments?: string;
    url?: string;
    headers?: string;
}

// Define submission status types
type SubmissionStatus = 'idle' | 'validating' | 'validationSuccess' | 'submittingApi' | 'error';

const CustomServiceModal: React.FC<CustomServiceModalProps> = ({ open, onClose, onCreateService }) => {
    const { toast } = useToast();
    const [submissionStatus, setSubmissionStatus] = useState<SubmissionStatus>('idle');
    const [serviceData, setServiceData] = useState<CustomServiceData>({
        name: '',
        type: 'streamableHttp',
        command: '',
        arguments: '',
        environments: '',
        url: '',
        headers: ''
    });
    const [errors, setErrors] = useState<Record<string, string>>({});

    useEffect(() => {
        if (open) {
            setServiceData({
                name: '',
                type: 'streamableHttp',
                command: '',
                arguments: '',
                environments: '',
                url: '',
                headers: ''
            });
            setErrors({});
            setSubmissionStatus('idle');
        } else {
            handleReset();
            setSubmissionStatus('idle');
        }
    }, [open]);

    const handleChange = (field: keyof CustomServiceData, value: string) => {
        setServiceData(prev => ({ ...prev, [field]: value }));
        if (errors[field]) {
            setErrors(prev => ({ ...prev, [field]: '' }));
        }
    };

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {};

        if (!serviceData.name.trim()) {
            newErrors.name = '服务名称不能为空';
        }

        if (serviceData.type === 'stdio') {
            if (!serviceData.command?.trim()) {
                newErrors.command = '命令不能为空';
            } else if (!serviceData.command.startsWith('npx ') && !serviceData.command.startsWith('uvx ')) {
                newErrors.command = '命令必须以 npx 或 uvx 开头';
            }
        } else if (serviceData.type === 'sse' || serviceData.type === 'streamableHttp') {
            if (!serviceData.url?.trim()) {
                newErrors.url = 'URL不能为空';
            } else {
                try {
                    new URL(serviceData.url);
                } catch (e) {
                    newErrors.url = '请输入有效的URL';
                }
            }
        }

        setErrors(newErrors);
        return Object.keys(newErrors).length === 0;
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        setSubmissionStatus('validating');

        if (!validateForm()) {
            setSubmissionStatus('error');
            return;
        }

        setSubmissionStatus('validationSuccess');
        await new Promise(resolve => setTimeout(resolve, 800));

        setSubmissionStatus('submittingApi');
        try {
            await onCreateService(serviceData);
            toast({
                title: '创建成功',
                description: `服务 ${serviceData.name} 已成功创建`
            });
            onClose();
        } catch (error: any) {
            toast({
                title: '创建失败',
                description: error.message || '未知错误',
                variant: 'destructive'
            });
            setSubmissionStatus('error');
        } finally {
            if (submissionStatus === 'submittingApi') {
            }
        }
    };

    const handleReset = () => {
        setServiceData({
            name: '',
            type: 'streamableHttp',
            command: '',
            arguments: '',
            environments: '',
            url: '',
            headers: ''
        });
        setErrors({});
    };

    const triggerCloseFromDialog = () => {
        onClose();
    };

    const isBusy = submissionStatus === 'validating' || submissionStatus === 'validationSuccess' || submissionStatus === 'submittingApi';

    return (
        <Dialog open={open} onOpenChange={(isOpen) => {
            if (!isOpen && !isBusy) {
                triggerCloseFromDialog();
            }
        }}>
            <DialogContent className="fixed left-[50%] top-[50%] z-50 grid w-full max-w-md translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%] sm:rounded-lg md:max-w-lg max-h-[90vh] overflow-y-auto">
                {isBusy && (
                    <div className="absolute inset-0 bg-black/80 backdrop-blur-sm flex flex-col items-center justify-center z-[60] rounded-lg">
                        <div className="bg-white/10 backdrop-blur-md rounded-xl p-8 flex flex-col items-center space-y-4 border border-white/20 shadow-2xl">
                            {submissionStatus === 'validating' && (
                                <>
                                    <Loader2 className="h-12 w-12 animate-spin text-blue-400" />
                                    <div className="text-center">
                                        <p className="text-white text-xl font-semibold">验证中</p>
                                        <p className="text-white/70 text-sm mt-1">正在检查表单数据...</p>
                                    </div>
                                </>
                            )}
                            {submissionStatus === 'validationSuccess' && (
                                <>
                                    <div className="relative">
                                        <CheckCircle className="h-12 w-12 text-green-400 animate-pulse" />
                                        <div className="absolute inset-0 h-12 w-12 bg-green-400/20 rounded-full animate-ping"></div>
                                    </div>
                                    <div className="text-center">
                                        <p className="text-white text-xl font-semibold">验证成功!</p>
                                        <p className="text-white/70 text-sm mt-1">正在创建服务...</p>
                                    </div>
                                </>
                            )}
                            {submissionStatus === 'submittingApi' && (
                                <>
                                    <Loader2 className="h-12 w-12 animate-spin text-purple-400" />
                                    <div className="text-center">
                                        <p className="text-white text-xl font-semibold">创建中</p>
                                        <p className="text-white/70 text-sm mt-1">正在保存服务配置...</p>
                                    </div>
                                </>
                            )}
                        </div>
                    </div>
                )}
                <DialogHeader>
                    <DialogTitle>创建自定义服务</DialogTitle>
                    <DialogDescription>
                        填写以下信息创建一个自定义MCP服务
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={handleSubmit} className="space-y-4 py-2">
                    <div className="space-y-2">
                        <Label htmlFor="service-name">服务名称</Label>
                        <Input
                            id="service-name"
                            value={serviceData.name}
                            onChange={(e) => handleChange('name', e.target.value)}
                            placeholder="输入服务名称"
                            className={errors.name ? 'border-red-500' : ''}
                        />
                        {errors.name && <p className="text-red-500 text-xs">{errors.name}</p>}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="service-type">服务类型</Label>
                        <Select
                            value={serviceData.type}
                            onValueChange={(value) => handleChange('type', value as any)}
                        >
                            <SelectTrigger id="service-type">
                                <SelectValue placeholder="选择服务类型" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="stdio">标准输入/输出 (stdio)</SelectItem>
                                <SelectItem value="sse">服务器发送事件 (sse)</SelectItem>
                                <SelectItem value="streamableHttp">可流式传输的HTTP (streamableHttp)</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>

                    {serviceData.type === 'stdio' && (
                        <>
                            <div className="space-y-2">
                                <Label htmlFor="service-command">命令</Label>
                                <Input
                                    id="service-command"
                                    value={serviceData.command}
                                    onChange={(e) => handleChange('command', e.target.value)}
                                    placeholder="输入命令 (npx 或 uvx)"
                                    className={errors.command ? 'border-red-500' : ''}
                                />
                                {errors.command && <p className="text-red-500 text-xs">{errors.command}</p>}
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="service-arguments">参数</Label>
                                <Textarea
                                    id="service-arguments"
                                    value={serviceData.arguments}
                                    onChange={(e) => handleChange('arguments', e.target.value)}
                                    placeholder="arg1&#10;arg2&#10;..."
                                    className="min-h-[80px]"
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="service-environments">环境变量</Label>
                                <Textarea
                                    id="service-environments"
                                    value={serviceData.environments}
                                    onChange={(e) => handleChange('environments', e.target.value)}
                                    placeholder="KEY1=value1&#10;KEY2=value2&#10;..."
                                    className="min-h-[80px]"
                                />
                            </div>
                        </>
                    )}

                    {(serviceData.type === 'sse' || serviceData.type === 'streamableHttp') && (
                        <>
                            <div className="space-y-2">
                                <Label htmlFor="service-url">URL</Label>
                                <Input
                                    id="service-url"
                                    value={serviceData.url}
                                    onChange={(e) => handleChange('url', e.target.value)}
                                    placeholder={
                                        serviceData.type === 'sse'
                                            ? "http://localhost/sse"
                                            : "http://localhost/mcp"
                                    }
                                    className={errors.url ? 'border-red-500' : ''}
                                />
                                {errors.url && <p className="text-red-500 text-xs">{errors.url}</p>}
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="service-headers">请求头</Label>
                                <Textarea
                                    id="service-headers"
                                    value={serviceData.headers}
                                    onChange={(e) => handleChange('headers', e.target.value)}
                                    placeholder="Content-Type=application/json&#10;Authorization=Bearer token&#10;..."
                                    className="min-h-[80px]"
                                />
                            </div>
                        </>
                    )}

                    <DialogFooter className="pt-4">
                        <Button
                            type="button"
                            variant="outline"
                            onClick={onClose}
                            disabled={isBusy}
                        >
                            取消
                        </Button>
                        <Button type="submit" disabled={isBusy}>
                            {submissionStatus === 'validating' && '验证中...'}
                            {submissionStatus === 'validationSuccess' && '验证成功'}
                            {submissionStatus === 'submittingApi' && '创建中...'}
                            {submissionStatus === 'idle' && '创建服务'}
                            {submissionStatus === 'error' && '创建服务'}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
};

export default CustomServiceModal; 