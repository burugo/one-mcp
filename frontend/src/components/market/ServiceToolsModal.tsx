import React, { useEffect, useState } from 'react';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from '@/components/ui/dialog';
import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion";
import { Badge } from '@/components/ui/badge';
import { Loader2 } from 'lucide-react';
import api, { APIResponse } from '@/utils/api';
import { useTranslation } from 'react-i18next';

interface ToolProperty {
    type?: string;
    description?: string;
    enum?: string[];
    default?: unknown;
}

interface ToolInputSchema {
    type?: string;
    properties?: Record<string, ToolProperty>;
    required?: string[];
}

interface Tool {
    name: string;
    description?: string;
    inputSchema?: ToolInputSchema;
}

interface ServiceToolsModalProps {
    serviceId: string;
    serviceName: string;
    serviceVersion?: string;
    isOpen: boolean;
    onClose: () => void;
}

const ServiceToolsModal: React.FC<ServiceToolsModalProps> = ({
    serviceId,
    serviceName,
    serviceVersion,
    isOpen,
    onClose,
}) => {
    const { t } = useTranslation();
    const [tools, setTools] = useState<Tool[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (isOpen && serviceId) {
            fetchTools();
        } else {
            setTools([]);
            setError(null);
        }
    }, [isOpen, serviceId]);

    const fetchTools = async () => {
        setLoading(true);
        setError(null);
        try {
            const response = await api.get(`/mcp_services/${serviceId}/tools`) as APIResponse<{ tools: Tool[] }>;
            if (response.success && response.data) {
                setTools(response.data.tools || []);
            } else {
                setError(response.message || 'Failed to fetch tools');
            }
        } catch (err: any) {
            console.error('Error fetching tools:', err);
            setError(err.message || 'An error occurred while fetching tools');
        } finally {
            setLoading(false);
        }
    };

    return (
        <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="sm:max-w-[600px] max-h-[80vh] flex flex-col">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2 flex-wrap">
                        {serviceName} - {t('serviceTools.toolsList')}
                        {serviceVersion && (
                            <Badge variant="outline" className="ml-1 font-normal">
                                {serviceVersion}
                            </Badge>
                        )}
                        {!loading && !error && (
                            <Badge variant="secondary" className="ml-2">
                                {tools.length}
                            </Badge>
                        )}
                    </DialogTitle>
                    <DialogDescription>
                        {t('serviceTools.availableToolsDescription')}
                    </DialogDescription>
                </DialogHeader>

                <div className="flex-1 min-h-[200px] max-h-[60vh] overflow-y-auto">
                    {loading ? (
                        <div className="flex justify-center items-center h-full text-muted-foreground">
                            <Loader2 className="h-8 w-8 animate-spin mr-2" />
                            <span>{t('common.loading')}</span>
                        </div>
                    ) : error ? (
                        <div className="flex justify-center items-center h-full text-destructive p-4 text-center">
                            <p>{error}</p>
                        </div>
                    ) : tools.length === 0 ? (
                        <div className="flex justify-center items-center h-full text-muted-foreground">
                            <p>{t('serviceTools.noToolsAvailable')}</p>
                        </div>
                    ) : (
                        <div className="pr-4">
                            <Accordion type="single" collapsible className="w-full">
                                {tools.map((tool, index) => (
                                    <AccordionItem key={index} value={`tool-${index}`}>
                                        <AccordionTrigger className="hover:no-underline hover:bg-muted/50 px-2 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background">
                                            <div className="flex flex-col items-start text-left">
                                                <span className="font-mono font-bold text-primary">{tool.name}</span>
                                                {tool.description && (
                                                    <span className="text-xs text-muted-foreground line-clamp-2 mt-1 font-normal">
                                                        {tool.description}
                                                    </span>
                                                )}
                                            </div>
                                        </AccordionTrigger>
                                        <AccordionContent className="px-4 py-2 bg-muted/30 rounded-b-md">
                                            <div className="space-y-3 max-h-[300px] overflow-y-auto">
                                                {tool.description && (
                                                    <div className="max-h-40 overflow-y-auto pr-1">
                                                        <h4 className="text-xs font-semibold mb-1 text-muted-foreground uppercase">{t('serviceTools.description')}</h4>
                                                        <p className="text-sm whitespace-pre-wrap break-words">{tool.description}</p>
                                                    </div>
                                                )}
                                                {tool.inputSchema?.properties && Object.keys(tool.inputSchema.properties).length > 0 && (
                                                    <div>
                                                        <h4 className="text-xs font-semibold mb-2 text-muted-foreground uppercase">{t('serviceTools.parameters')}</h4>
                                                        <table className="w-full text-sm border-collapse">
                                                            <thead>
                                                                <tr className="border-b border-border">
                                                                    <th className="text-left py-1.5 pr-2 font-medium text-muted-foreground">{t('serviceTools.parameterName')}</th>
                                                                    <th className="text-left py-1.5 pr-2 font-medium text-muted-foreground">{t('serviceTools.parameterType')}</th>
                                                                    <th className="text-left py-1.5 font-medium text-muted-foreground">{t('serviceTools.parameterDescription')}</th>
                                                                </tr>
                                                            </thead>
                                                            <tbody>
                                                                {Object.entries(tool.inputSchema.properties).map(([name, prop]) => (
                                                                    <tr key={name} className="border-b border-muted last:border-0">
                                                                        <td className="py-1.5 pr-2 font-mono text-primary">
                                                                            {name}
                                                                            {tool.inputSchema?.required?.includes(name) && (
                                                                                <span className="text-destructive ml-0.5">*</span>
                                                                            )}
                                                                        </td>
                                                                        <td className="py-1.5 pr-2 text-muted-foreground">
                                                                            {prop.type || '-'}
                                                                        </td>
                                                                        <td className="py-1.5 text-foreground/80">
                                                                            {prop.description || '-'}
                                                                        </td>
                                                                    </tr>
                                                                ))}
                                                            </tbody>
                                                        </table>
                                                    </div>
                                                )}
                                            </div>
                                        </AccordionContent>
                                    </AccordionItem>
                                ))}
                            </Accordion>
                        </div>
                    )}
                </div>
            </DialogContent>
        </Dialog>
    );
};

export default ServiceToolsModal;
