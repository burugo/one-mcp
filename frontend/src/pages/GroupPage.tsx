import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { Plus, Pencil, Trash2, Copy, Layers, Download } from 'lucide-react';
import api, { GroupService } from '@/utils/api';
import { useServerAddress } from '@/hooks/useServerAddress';
import { copyToClipboard } from '@/utils/clipboard';
import { useAuth } from '@/contexts/AuthContext';

interface MCPService {
    id: number;
    name: string;
    display_name: string;
    description?: string;
    enabled?: boolean;
}

interface Group {
    id: number;
    name: string;
    display_name: string;
    description: string;
    service_ids_json: string;
    enabled: boolean;
}

interface GroupModalProps {
    isOpen: boolean;
    onClose: () => void;
    group: Group | null;
    services: MCPService[];
    onSave: (data: any) => Promise<void>;
}

const GroupModal: React.FC<GroupModalProps> = ({ isOpen, onClose, group, services, onSave }) => {
    const { t } = useTranslation();
    const [formData, setFormData] = useState({
        name: '',
        display_name: '',
        service_ids: [] as number[],
        enabled: true
    });
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (group) {
            let ids: number[] = [];
            try {
                ids = JSON.parse(group.service_ids_json || '[]');
            } catch (e) {
                console.error('Failed to parse service IDs', e);
            }
            // Filter out disabled services (only keep IDs that exist in enabled services list)
            const enabledServiceIds = new Set(services.map(s => s.id));
            ids = ids.filter(id => enabledServiceIds.has(id));
            
            setFormData({
                name: group.name,
                display_name: group.display_name,
                service_ids: ids,
                enabled: group.enabled
            });
        } else {
            setFormData({
                name: '',
                display_name: '',
                service_ids: [],
                enabled: true
            });
        }
    }, [group, isOpen, services]);

    // Generate description automatically based on selected services
    const generateDescription = (): string => {
        const selectedServices = services.filter(svc => formData.service_ids.includes(svc.id));
        if (selectedServices.length === 0) return '';
        
        const lines = selectedServices.map(svc => {
            const desc = svc.description || svc.display_name || svc.name;
            return `- ${svc.name}: ${desc}`;
        });
        return `This group contains the following MCP services:\n${lines.join('\n')}`;
    };

    const handleSubmit = async () => {
        if (!formData.name || !formData.display_name) {
            return;
        }
        setLoading(true);
        try {
            const description = generateDescription();
            await onSave({
                ...formData,
                description,
                service_ids_json: JSON.stringify(formData.service_ids)
            });
            onClose();
        } catch (error) {
            console.error(error);
        } finally {
            setLoading(false);
        }
    };

    const toggleService = (id: number) => {
        setFormData(prev => {
            const ids = prev.service_ids.includes(id)
                ? prev.service_ids.filter(sid => sid !== id)
                : [...prev.service_ids, id];
            return { ...prev, service_ids: ids };
        });
    };

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="max-w-2xl">
                <DialogHeader>
                    <DialogTitle>{group ? t('groups.edit') : t('groups.create')}</DialogTitle>
                    <DialogDescription>{t('groups.description')}</DialogDescription>
                </DialogHeader>
                <div className="grid gap-4 py-4">
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="name" className="text-right">
                            {t('groups.name')}
                        </Label>
                        <Input
                            id="name"
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                            className="col-span-3"
                            disabled={!!group} // Name is ID, tricky to change if used in URL
                        />
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="displayName" className="text-right">
                            {t('groups.displayName')}
                        </Label>
                        <Input
                            id="displayName"
                            value={formData.display_name}
                            onChange={(e) => setFormData({ ...formData, display_name: e.target.value })}
                            className="col-span-3"
                        />
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label className="text-right">{t('groups.enabled')}</Label>
                        <Switch
                            checked={formData.enabled}
                            onCheckedChange={(checked) => setFormData({ ...formData, enabled: checked })}
                        />
                    </div>
                    <div className="grid grid-cols-4 gap-4">
                        <Label className="text-right pt-2">{t('groups.services')}</Label>
                        <div className="col-span-3 border rounded-md p-4 max-h-60 overflow-y-auto space-y-3">
                            {services.length === 0 ? (
                                <div className="text-muted-foreground text-sm">{t('groups.noServices')}</div>
                            ) : (
                                services.map(svc => (
                                    <div key={svc.id} className="flex items-center space-x-3 py-1">
                                        <Switch
                                            id={`svc-${svc.id}`}
                                            checked={formData.service_ids.includes(svc.id)}
                                            onCheckedChange={() => toggleService(svc.id)}
                                        />
                                        <label 
                                            htmlFor={`svc-${svc.id}`}
                                            className="text-sm font-medium leading-none cursor-pointer"
                                        >
                                            {svc.display_name || svc.name}
                                            <span className="text-muted-foreground ml-2 font-normal">({svc.name})</span>
                                        </label>
                                    </div>
                                ))
                            )}
                        </div>
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={onClose}>{t('common.cancel')}</Button>
                    <Button onClick={handleSubmit} disabled={loading}>{t('common.save')}</Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
};

export const GroupPage = () => {
    const { t } = useTranslation();
    const { toast } = useToast();
    const serverAddress = useServerAddress();
    const { currentUser, updateUserInfo } = useAuth();
    const [groups, setGroups] = useState<Group[]>([]);
    const [services, setServices] = useState<MCPService[]>([]);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [editingGroup, setEditingGroup] = useState<Group | null>(null);
    const [userToken, setUserToken] = useState<string>('');

    // Fetch user token
    useEffect(() => {
        const fetchUserToken = async () => {
            try {
                // First check if currentUser already has token
                if (currentUser?.token) {
                    setUserToken(currentUser.token);
                    return;
                }

                // If not, fetch from API
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
                        // Update AuthContext
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

        if (currentUser) {
            fetchUserToken();
        }
    }, [currentUser, updateUserInfo]);

    const fetchData = async () => {
        try {
            const [groupsResp, servicesResp] = await Promise.all([
                GroupService.getAll(),
                api.get<MCPService[]>('/mcp_market/installed?enabled=true')
            ]);
            
            if (groupsResp.success) {
                setGroups(groupsResp.data || []);
            }
            if (servicesResp.success) {
                setServices(servicesResp.data || []);
            }
        } catch (error) {
            console.error('Failed to fetch data', error);
            toast({
                variant: "destructive",
                title: t('common.error'),
                description: t('dashboard.fetchDataFailed')
            });
        }
    };

    useEffect(() => {
        fetchData();
    }, []);

    const handleCreate = () => {
        setEditingGroup(null);
        setIsModalOpen(true);
    };

    const handleEdit = (group: Group) => {
        setEditingGroup(group);
        setIsModalOpen(true);
    };

    const handleDelete = async (group: Group) => {
        if (!confirm(t('groups.deleteConfirmDesc', { name: group.name }))) return;
        
        try {
            const resp = await GroupService.delete(group.id);
            if (resp.success) {
                toast({ title: t('common.success'), description: t('common.success') });
                fetchData();
            }
        } catch (error) {
            console.error(error);
            toast({ variant: "destructive", title: t('common.error'), description: t('common.error') });
        }
    };

    const handleSave = async (data: any) => {
        try {
            let resp;
            if (editingGroup) {
                resp = await GroupService.update(editingGroup.id, data);
            } else {
                resp = await GroupService.create(data);
            }
            
            if (resp.success) {
                toast({ title: t('common.success'), description: t('common.success') });
                fetchData();
            } else {
                toast({ variant: "destructive", title: t('common.error'), description: resp.message });
            }
        } catch (error) {
            throw error;
        }
    };

    const getGroupUrl = (name: string) => {
        const baseUrl = serverAddress.endsWith('/') ? serverAddress.slice(0, -1) : serverAddress;
        return `${baseUrl}/group/${name}/mcp?key=${userToken || '<YOUR_TOKEN>'}`;
    };

    const handleCopyToClipboard = async (text: string) => {
        const result = await copyToClipboard(text);
        if (result.success) {
            toast({
                title: t('common.success'),
                description: t('services.copiedToClipboard'),
            });
        } else {
            toast({
                variant: "destructive",
                title: t('common.error'),
                description: t('clipboardError.execCommandFailed'),
            });
        }
    };

    const handleExportSkill = async (group: Group) => {
        try {
            const response = await fetch(`/api/groups/${group.id}/export`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('token')}`,
                },
            });
            
            if (!response.ok) {
                throw new Error('Export failed');
            }
            
            // Get filename from Content-Disposition header or use default
            const contentDisposition = response.headers.get('Content-Disposition');
            let filename = `one-mcp-${group.name.replace(/_/g, '-')}.zip`;
            if (contentDisposition) {
                const match = contentDisposition.match(/filename=(.+)/);
                if (match) {
                    filename = match[1];
                }
            }
            
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
            
            toast({
                title: t('common.success'),
                description: t('groups.exportSuccess'),
            });
        } catch (error) {
            console.error('Export failed', error);
            toast({
                variant: "destructive",
                title: t('common.error'),
                description: t('groups.exportFailed'),
            });
        }
    };

    const handleToggleEnabled = async (group: Group) => {
        try {
            const resp = await GroupService.update(group.id, {
                ...group,
                enabled: !group.enabled
            });
            if (resp.success) {
                fetchData();
            }
        } catch (error) {
            console.error(error);
            toast({ variant: "destructive", title: t('common.error'), description: t('common.error') });
        }
    };

    return (
        <TooltipProvider>
            <div className="container mx-auto p-6 space-y-6">
                <div className="flex justify-between items-center">
                    <div>
                        <h1 className="text-3xl font-bold tracking-tight">{t('groups.title')}</h1>
                        <p className="text-muted-foreground mt-2">{t('groups.description')}</p>
                    </div>
                    <Button onClick={handleCreate}>
                        <Plus className="mr-2 h-4 w-4" />
                        {t('groups.create')}
                    </Button>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    {groups.map(group => {
                        const url = getGroupUrl(group.name);
                        // Count only enabled services that exist in the services list
                        let serviceCount = 0;
                        try {
                            const groupServiceIds = JSON.parse(group.service_ids_json || '[]') as number[];
                            const enabledServiceIds = new Set(services.map(s => s.id));
                            serviceCount = groupServiceIds.filter(id => enabledServiceIds.has(id)).length;
                        } catch {}

                        return (
                            <Card key={group.id} className="border-border shadow-sm hover:shadow transition-shadow duration-200 bg-card/30 flex flex-col">
                                <CardHeader>
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center">
                                            <div className="bg-primary/10 p-2 rounded-md mr-3">
                                                <Layers className="w-6 h-6 text-primary" />
                                            </div>
                                            <div>
                                                <CardTitle className="text-lg">{group.display_name}</CardTitle>
                                                <p className="font-mono text-xs text-muted-foreground">{group.name}</p>
                                            </div>
                                        </div>
                                        <div className="flex items-center space-x-1 ml-2">
                                            <button
                                                className="p-1 rounded hover:bg-blue-100 text-blue-500"
                                                onClick={() => handleEdit(group)}
                                                title={t('common.edit')}
                                            >
                                                <Pencil size={16} />
                                            </button>
                                            <button
                                                className="p-1 rounded hover:bg-red-100 text-red-500"
                                                onClick={() => handleDelete(group)}
                                                title={t('common.delete')}
                                            >
                                                <Trash2 size={18} />
                                            </button>
                                        </div>
                                    </div>
                                </CardHeader>
                                <CardContent className="flex-grow space-y-4">
                                    <Tooltip>
                                        <TooltipTrigger asChild>
                                            <p className="text-sm text-muted-foreground line-clamp-2 cursor-help">
                                                {group.description || t('groups.description')}
                                            </p>
                                        </TooltipTrigger>
                                        <TooltipContent side="bottom" className="max-w-sm whitespace-pre-wrap">
                                            <p>{group.description || t('groups.description')}</p>
                                        </TooltipContent>
                                    </Tooltip>
                                    
                                    <div className="flex items-center gap-2 text-sm">
                                        <Layers className="h-4 w-4 text-muted-foreground" />
                                        <span>{serviceCount} {t('groups.services')}</span>
                                    </div>

                                    <div className="bg-muted p-3 rounded-md space-y-2">
                                        <div className="text-xs font-medium text-muted-foreground">{t('groups.endpoint')}</div>
                                        <div className="flex items-center gap-2">
                                            <code className="text-xs flex-1 truncate bg-background p-1.5 rounded border">
                                                {url}
                                            </code>
                                            <Button 
                                                variant="ghost" 
                                                size="icon" 
                                                className="h-8 w-8 shrink-0"
                                            onClick={() => handleCopyToClipboard(url)}
                                            >
                                                <Copy className="h-4 w-4" />
                                            </Button>
                                        </div>
                                    </div>
                                </CardContent>
                                <CardFooter className="flex justify-between items-end mt-auto">
                                    <Button variant="outline" size="sm" className="h-6" onClick={() => handleExportSkill(group)}>
                                        <Download className="mr-1 h-3 w-3" />
                                        {t('groups.exportSkill')}
                                    </Button>
                                    <Switch
                                        checked={group.enabled}
                                        onCheckedChange={() => handleToggleEnabled(group)}
                                    />
                                </CardFooter>
                            </Card>
                        );
                    })}
                </div>

                <GroupModal 
                    isOpen={isModalOpen} 
                    onClose={() => setIsModalOpen(false)} 
                    group={editingGroup}
                    services={services}
                    onSave={handleSave}
                />
            </div>
        </TooltipProvider>
    );
};

export default GroupPage;
