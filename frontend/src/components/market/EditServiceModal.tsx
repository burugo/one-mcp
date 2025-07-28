import React, { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { useTranslation } from 'react-i18next';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { CheckCircle, Loader2 } from 'lucide-react';
import { ServiceType } from '@/store/marketStore';

interface EditServiceModalProps {
    open: boolean;
    onClose: () => void;
    service: ServiceType;
    onUpdateService: (serviceData: EditServiceData) => Promise<void>;
}

export interface EditServiceData {
    id: string;
    name?: string;
    display_name: string;
    description: string;
    type: 'stdio' | 'sse' | 'streamableHttp';
    command?: string;
    arguments?: string;
    url?: string;
    headers?: string;
    envVars?: string;
    commandLine?: string; // For stdio services merged command display
}

// Define submission status types
type SubmissionStatus = 'idle' | 'validating' | 'validationSuccess' | 'submittingApi' | 'error';

const EditServiceModal: React.FC<EditServiceModalProps> = ({ open, onClose, service, onUpdateService }) => {
    const { t } = useTranslation();
    const { toast } = useToast();
    const [submissionStatus, setSubmissionStatus] = useState<SubmissionStatus>('idle');
    const [serviceData, setServiceData] = useState<EditServiceData>({
        id: '',
        name: '',
        display_name: '',
        description: '',
        type: 'streamableHttp',
        command: '',
        arguments: '',
        url: '',
        headers: '',
        envVars: ''
    });
    const [errors, setErrors] = useState<Record<string, string>>({});

    // Initialize form data when service prop changes
    useEffect(() => {
        if (service && open) {
            // Determine service type based on service.type or command field
            let serviceType: 'stdio' | 'sse' | 'streamableHttp' = 'streamableHttp';

            // Try to determine type from the service data
            if (service.type) {
                // Use the explicit type if available (most reliable)
                serviceType = service.type;
            } else if (service.command) {
                // If no explicit type, determine from command field
                const command = service.command.trim();

                // Check if it's a stdio command (npx, uvx)
                if (command === 'npx' || command === 'uvx') {
                    serviceType = 'stdio';
                } else {
                    // Check if it looks like a URL (for SSE/HTTP services)
                    try {
                        new URL(command);
                        // It's a valid URL, determine if SSE or streamableHttp
                        if (command.includes('/sse')) {
                            serviceType = 'sse';
                        } else {
                            serviceType = 'streamableHttp';
                        }
                    } catch {
                        // Not a valid URL, fallback to source-based detection
                        if (service.source === 'npm' || service.source === 'pypi') {
                            serviceType = 'stdio';
                        } else {
                            serviceType = 'streamableHttp'; // Default assumption
                        }
                    }
                }
            } else if (service.source === 'npm' || service.source === 'pypi') {
                // Fallback to source-based detection if no command
                serviceType = 'stdio';
            }

            // Extract data based on service type
            let url = '';
            let headers = '';
            let args = '';
            let envVars = '';

            if (serviceType === 'sse' || serviceType === 'streamableHttp') {
                // For SSE/HTTP services, URL is stored in the command field
                url = service.command || '';

                // Extract headers from headers_json if available
                if (service.headers_json) {
                    try {
                        const headersObj = JSON.parse(service.headers_json);
                        headers = Object.entries(headersObj)
                            .map(([key, value]) => `${key}=${value}`)
                            .join('\n');
                    } catch (e) {
                        console.warn('Failed to parse headers_json:', e);
                    }
                }
            } else if (serviceType === 'stdio') {
                // For stdio services, create merged command line display
                let commandLine = service.command || '';
                if (service.args_json) {
                    try {
                        const argsArray = JSON.parse(service.args_json);
                        if (Array.isArray(argsArray)) {
                            // Merge command and arguments into single command line
                            commandLine = [service.command || '', ...argsArray].filter(Boolean).join(' ');
                            // Also keep separate args for backward compatibility
                            args = argsArray.join('\n');
                        }
                    } catch (e) {
                        console.warn('Failed to parse args_json:', e);
                    }
                }
                // Store the merged command line in url field (reusing existing field)
                url = commandLine;

                // Extract environment variables from default_envs_json
                if (service.default_envs_json) {
                    try {
                        const envsObj = JSON.parse(service.default_envs_json);
                        envVars = Object.entries(envsObj)
                            .map(([key, value]) => `${key}=${value}`)
                            .join('\n');
                    } catch (e) {
                        console.warn('Failed to parse default_envs_json:', e);
                    }
                }
            }

            setServiceData({
                id: service.id,
                name: service.name,
                display_name: service.display_name || service.name,
                description: service.description || '',
                type: serviceType,
                command: service.command || '',
                arguments: args,
                url: url,
                headers: headers,
                envVars: envVars,
                commandLine: serviceType === 'stdio' ? url : undefined
            });
            setErrors({});
            setSubmissionStatus('idle');
        }
    }, [service, open]);

    useEffect(() => {
        if (!open) {
            setSubmissionStatus('idle');
        }
    }, [open]);

    const handleChange = (field: keyof EditServiceData, value: string) => {
        setServiceData(prev => ({ ...prev, [field]: value }));
        if (errors[field]) {
            setErrors(prev => ({ ...prev, [field]: '' }));
        }
    };

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {};

        if (!serviceData.display_name.trim()) {
            newErrors.display_name = 'Display name cannot be empty';
        }

        if (serviceData.type === 'sse' || serviceData.type === 'streamableHttp') {
            if (!serviceData.url?.trim()) {
                newErrors.url = 'URL cannot be empty';
            } else {
                try {
                    new URL(serviceData.url);
                } catch {
                    newErrors.url = 'Please enter a valid URL';
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
            await onUpdateService(serviceData);
            setSubmissionStatus('idle');
        } catch (error: unknown) {
            let errorMessage = 'Unknown error';

            if (error &&
                typeof error === 'object' &&
                'response' in error) {
                const axiosError = error as {
                    response?: {
                        data?: { message?: string }
                    }
                };

                if (axiosError.response?.data?.message) {
                    errorMessage = axiosError.response.data.message;
                }
            } else if (error instanceof Error) {
                errorMessage = error.message;
            }

            toast({
                title: 'Update Failed',
                description: errorMessage,
                variant: 'destructive'
            });
            setSubmissionStatus('idle');
        }
    };

    const triggerCloseFromDialog = () => {
        onClose();
    };

    const isBusy = submissionStatus === 'validating' || submissionStatus === 'validationSuccess' || submissionStatus === 'submittingApi';

    return (
        <Dialog open={open} onOpenChange={(isOpen: boolean) => {
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
                                        <p className="text-white text-xl font-semibold">{t('customServiceModal.status.validating')}</p>
                                        <p className="text-white/70 text-sm mt-1">{t('customServiceModal.status.validatingDescription')}</p>
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
                                        <p className="text-white text-xl font-semibold">{t('customServiceModal.status.validationSuccess')}</p>
                                        <p className="text-white/70 text-sm mt-1">{t('editServiceModal.status.validationSuccessDescription')}</p>
                                    </div>
                                </>
                            )}
                            {submissionStatus === 'submittingApi' && (
                                <>
                                    <Loader2 className="h-12 w-12 animate-spin text-purple-400" />
                                    <div className="text-center">
                                        <p className="text-white text-xl font-semibold">{t('editServiceModal.status.updating')}</p>
                                        <p className="text-white/70 text-sm mt-1">{t('editServiceModal.status.updatingDescription')}</p>
                                    </div>
                                </>
                            )}
                        </div>
                    </div>
                )}
                <DialogHeader>
                    <DialogTitle>{t('editServiceModal.title')}</DialogTitle>
                    <DialogDescription>
                        {t('editServiceModal.description')}
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={handleSubmit} className="space-y-4 py-2">
                    <div className="space-y-2">
                        <Label htmlFor="service-display-name">{t('editServiceModal.form.displayName')}</Label>
                        <Input
                            id="service-display-name"
                            value={serviceData.display_name}
                            onChange={(e) => handleChange('display_name', e.target.value)}
                            placeholder={t('editServiceModal.form.displayNamePlaceholder')}
                            className={errors.display_name ? 'border-red-500' : ''}
                        />
                        {errors.display_name && <p className="text-red-500 text-xs">{errors.display_name}</p>}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="service-description">{t('editServiceModal.form.description')}</Label>
                        <Textarea
                            id="service-description"
                            value={serviceData.description}
                            onChange={(e) => handleChange('description', e.target.value)}
                            placeholder={t('editServiceModal.form.descriptionPlaceholder')}
                            className="min-h-[80px]"
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="service-type">{t('customServiceModal.form.serviceType')}</Label>
                        <Select
                            value={serviceData.type}
                            onValueChange={(value) => handleChange('type', value as any)}
                            disabled={true}
                        >
                            <SelectTrigger id="service-type" className="opacity-50">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="stdio">{t('customServiceModal.serviceTypes.stdio')}</SelectItem>
                                <SelectItem value="sse">{t('customServiceModal.serviceTypes.sse')}</SelectItem>
                                <SelectItem value="streamableHttp">{t('customServiceModal.serviceTypes.streamableHttp')}</SelectItem>
                            </SelectContent>
                        </Select>
                        <p className="text-xs text-muted-foreground">{t('editServiceModal.form.serviceTypeNote')}</p>
                    </div>

                    {serviceData.type === 'stdio' && (
                        <>
                            <div className="space-y-2">
                                <Label htmlFor="service-command-line">{t('editServiceModal.form.commandLine')}</Label>
                                <Input
                                    id="service-command-line"
                                    value={serviceData.commandLine || ''}
                                    readOnly
                                    className="bg-gray-50 dark:bg-gray-800 opacity-75"
                                    placeholder={t('editServiceModal.form.commandLinePlaceholder')}
                                />
                                <p className="text-xs text-muted-foreground">{t('editServiceModal.form.commandLineNote')}</p>
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="service-env-vars">{t('customServiceModal.form.environments')}</Label>
                                <Textarea
                                    id="service-env-vars"
                                    value={serviceData.envVars}
                                    onChange={(e) => handleChange('envVars', e.target.value)}
                                    placeholder={t('customServiceModal.form.environmentsPlaceholder')}
                                    className="min-h-[100px]"
                                />
                                <p className="text-xs text-muted-foreground">{t('editServiceModal.form.environmentsNote')}</p>
                            </div>
                        </>
                    )}

                    {(serviceData.type === 'sse' || serviceData.type === 'streamableHttp') && (
                        <>
                            <div className="space-y-2">
                                <Label htmlFor="service-url">{t('customServiceModal.form.serverUrl')}</Label>
                                <Input
                                    id="service-url"
                                    value={serviceData.url}
                                    onChange={(e) => handleChange('url', e.target.value)}
                                    placeholder={t('customServiceModal.form.serverUrlPlaceholder')}
                                    className={errors.url ? 'border-red-500' : ''}
                                />
                                {errors.url && <p className="text-red-500 text-xs">{errors.url}</p>}
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="service-headers">{t('customServiceModal.form.requestHeaders')}</Label>
                                <Textarea
                                    id="service-headers"
                                    value={serviceData.headers}
                                    onChange={(e) => handleChange('headers', e.target.value)}
                                    placeholder={t('customServiceModal.form.requestHeadersPlaceholder')}
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
                            {t('customServiceModal.actions.cancel')}
                        </Button>
                        <Button type="submit" disabled={isBusy}>
                            {submissionStatus === 'validating' && t('editServiceModal.actions.validating')}
                            {submissionStatus === 'validationSuccess' && t('editServiceModal.actions.validationSuccess')}
                            {submissionStatus === 'submittingApi' && t('editServiceModal.actions.updating')}
                            {(submissionStatus === 'idle' || submissionStatus === 'error') && t('editServiceModal.actions.saveChanges')}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
};

export default EditServiceModal; 