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

interface CustomServiceModalProps {
    open: boolean;
    onClose: () => void;
    onCreateService: (serviceData: CustomServiceData) => Promise<void>;
    autoFillEnv: string;
    setAutoFillEnv: (value: string) => void;
}

export interface CustomServiceData {
    name: string;
    type: 'stdio' | 'sse' | 'streamableHttp';
    command?: string;
    environments?: string;
    url?: string;
    headers?: string;
}

// Define submission status types
type SubmissionStatus = 'idle' | 'validating' | 'validationSuccess' | 'submittingApi' | 'error';

const CustomServiceModal: React.FC<CustomServiceModalProps> = ({ open, onClose, onCreateService, autoFillEnv, setAutoFillEnv }) => {
    const { t } = useTranslation();
    const { toast } = useToast();
    const [submissionStatus, setSubmissionStatus] = useState<SubmissionStatus>('idle');
    const [serviceData, setServiceData] = useState<CustomServiceData>({
        name: '',
        type: 'streamableHttp',
        command: '',
        environments: '',
        url: '',
        headers: ''
    });
    const [errors, setErrors] = useState<Record<string, string>>({});

    useEffect(() => {
        if (autoFillEnv) {
            // Extract the environment variable name from autoFillEnv (e.g., "GITHUB_PERSONAL_ACCESS_TOKEN=" -> "GITHUB_PERSONAL_ACCESS_TOKEN")
            const envVarName = autoFillEnv.split('=')[0];

            setServiceData(prev => {
                // Parse existing environment variables
                const existingEnvs = prev.environments ? prev.environments.split('\n').filter(line => line.trim()) : [];

                // Check if the environment variable already exists
                const envVarExists = existingEnvs.some(line => {
                    const existingVarName = line.split('=')[0];
                    return existingVarName === envVarName;
                });

                // Only add if it doesn't exist
                if (!envVarExists) {
                    const newEnvironments = prev.environments ? `${prev.environments}\n${autoFillEnv}` : autoFillEnv;
                    return {
                        ...prev,
                        environments: newEnvironments
                    };
                }

                // If it already exists, don't modify the environments
                return prev;
            });
            setAutoFillEnv(''); // Reset after processing
        }
    }, [autoFillEnv, setAutoFillEnv]);

    useEffect(() => {
        if (open) {
            setServiceData({
                name: '',
                type: 'streamableHttp',
                command: '',
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
            newErrors.name = t('customServiceModal.form.serviceNamePlaceholder');
        }

        if (serviceData.type === 'stdio') {
            if (!serviceData.command?.trim()) {
                newErrors.command = t('customServiceModal.form.commandPlaceholder');
            } else {
                const trimmedCommand = serviceData.command.trim();
                const isValidCommand = trimmedCommand.startsWith('npx ') ||
                    trimmedCommand.startsWith('uvx ') ||
                    trimmedCommand === 'npx' ||
                    trimmedCommand === 'uvx';

                if (!isValidCommand) {
                    newErrors.command = t('customServiceModal.messages.commandMustStartWith');
                }
            }
        } else if (serviceData.type === 'sse' || serviceData.type === 'streamableHttp') {
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
            await onCreateService(serviceData);
            // Remove onSuccess() call - let the parent component handle all success logic
            setSubmissionStatus('idle'); // Reset to idle after successful submission
        } catch (error: unknown) {
            // Extract the actual error message from the API response
            let errorMessage = t('customServiceModal.messages.unknownError');

            // Type-safe error handling for axios-like error objects
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
                title: t('customServiceModal.messages.createFailed'),
                description: errorMessage,
                variant: 'destructive'
            });
            setSubmissionStatus('idle'); // Reset to idle so user can retry
        } finally {
            // No need for additional logic in finally block
        }
    };

    const handleReset = () => {
        setServiceData({
            name: '',
            type: 'streamableHttp',
            command: '',
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
                                        <p className="text-white/70 text-sm mt-1">{t('customServiceModal.status.validationSuccessDescription')}</p>
                                    </div>
                                </>
                            )}
                            {submissionStatus === 'submittingApi' && (
                                <>
                                    <Loader2 className="h-12 w-12 animate-spin text-purple-400" />
                                    <div className="text-center">
                                        <p className="text-white text-xl font-semibold">{t('customServiceModal.status.creating')}</p>
                                        <p className="text-white/70 text-sm mt-1">{t('customServiceModal.status.creatingDescription')}</p>
                                    </div>
                                </>
                            )}
                        </div>
                    </div>
                )}
                <DialogHeader>
                    <DialogTitle>{t('customServiceModal.title')}</DialogTitle>
                    <DialogDescription>
                        {t('customServiceModal.description')}
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={handleSubmit} className="space-y-4 py-2">
                    <div className="space-y-2">
                        <Label htmlFor="service-name">{t('customServiceModal.form.serviceName')}</Label>
                        <Input
                            id="service-name"
                            value={serviceData.name}
                            onChange={(e) => handleChange('name', e.target.value)}
                            placeholder={t('customServiceModal.form.serviceNamePlaceholder')}
                            className={errors.name ? 'border-red-500' : ''}
                        />
                        {errors.name && <p className="text-red-500 text-xs">{errors.name}</p>}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="service-type">{t('customServiceModal.form.serviceType')}</Label>
                        <Select
                            value={serviceData.type}
                            onValueChange={(value) => handleChange('type', value as any)}
                        >
                            <SelectTrigger id="service-type">
                                <SelectValue placeholder={t('customServiceModal.form.serviceTypePlaceholder')} />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="stdio">{t('customServiceModal.serviceTypes.stdio')}</SelectItem>
                                <SelectItem value="sse">{t('customServiceModal.serviceTypes.sse')}</SelectItem>
                                <SelectItem value="streamableHttp">{t('customServiceModal.serviceTypes.streamableHttp')}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>

                    {serviceData.type === 'stdio' && (
                        <>
                            <div className="space-y-2">
                                <Label htmlFor="service-command">{t('customServiceModal.form.command')}</Label>
                                <Input
                                    id="service-command"
                                    value={serviceData.command}
                                    onChange={(e) => handleChange('command', e.target.value)}
                                    placeholder={t('customServiceModal.form.commandPlaceholder')}
                                    className={errors.command ? 'border-red-500' : ''}
                                />
                                {errors.command && <p className="text-red-500 text-xs">{errors.command}</p>}
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="service-environments">{t('customServiceModal.form.environments')}</Label>
                                <Textarea
                                    id="service-environments"
                                    value={serviceData.environments}
                                    onChange={(e) => handleChange('environments', e.target.value)}
                                    placeholder={t('customServiceModal.form.environmentsPlaceholder')}
                                    className="min-h-[80px]"
                                />
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
                            {submissionStatus === 'validating' && t('customServiceModal.actions.validating')}
                            {submissionStatus === 'validationSuccess' && t('customServiceModal.actions.validationSuccess')}
                            {submissionStatus === 'submittingApi' && t('customServiceModal.actions.creating')}
                            {submissionStatus === 'idle' && t('customServiceModal.actions.createService')}
                            {submissionStatus === 'error' && t('customServiceModal.actions.createService')}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
};

export default CustomServiceModal; 