import React, { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Copy, Check } from 'lucide-react';

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

    React.useEffect(() => {
        setEnvValues(getEnvVars(service));
    }, [service]);

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

    const handleCopy = async (label: string, value: string) => {
        try {
            await navigator.clipboard.writeText(value);
            setCopied((prev) => ({ ...prev, [label]: true }));
            setTimeout(() => setCopied((prev) => ({ ...prev, [label]: false })), 1200);
        } catch { }
    };

    // 生成 endpoint
    const sseEndpoint = `/api/sse/${service?.name || ''}`;
    const httpEndpoint = `/api/http/${service?.name || ''}`;

    return (
        <Dialog open={open} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader className="mb-4">
                    <DialogTitle>Service Configuration</DialogTitle>
                    <DialogDescription>
                        Adjust the settings for this service. Click save when you're done.
                    </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 mt-2">
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
                </div>
                <div className="mt-6 space-y-3">
                    <div className="flex items-center gap-2">
                        <span className="w-28 text-sm font-medium">SSE Endpoint</span>
                        <Input value={sseEndpoint} readOnly className="flex-1" />
                        <Button
                            size="icon"
                            variant="ghost"
                            onClick={() => handleCopy('sse', sseEndpoint)}
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
                            onClick={() => handleCopy('http', httpEndpoint)}
                        >
                            {copied['http'] ? <Check className="text-green-500 w-5 h-5" /> : <Copy className="w-5 h-5" />}
                        </Button>
                    </div>
                </div>
                <DialogFooter className="mt-4">
                    <Button variant="outline" onClick={onClose} type="button">Cancel</Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
};

export default ServiceConfigModal; 