import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { Plus, Edit2, Trash2, Copy, Layers, ExternalLink } from 'lucide-react';
import api, { GroupService } from '@/utils/api';
import { useServerAddress } from '@/hooks/useServerAddress';
import { writeToClipboard } from '@/utils/clipboard';

interface MCPService {
    ID: number;
    name: string;
    display_name: string;
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
        description: '',
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
            setFormData({
                name: group.name,
                display_name: group.display_name,
                description: group.description,
                service_ids: ids,
                enabled: group.enabled
            });
        } else {
            setFormData({
                name: '',
                display_name: '',
                description: '',
                service_ids: [],
                enabled: true
            });
        }
    }, [group, isOpen]);

    const handleSubmit = async () => {
        if (!formData.name || !formData.display_name) {
            return;
        }
        setLoading(true);
        try {
            await onSave({
                ...formData,
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
                        <Label htmlFor="description" className="text-right">
                            {t('groups.desc')}
                        </Label>
                        <Textarea
                            id="description"
                            value={formData.description}
                            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
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
                        <div className="col-span-3 border rounded-md p-4 max-h-60 overflow-y-auto space-y-2">
                            {services.length === 0 ? (
                                <div className="text-muted-foreground text-sm">{t('groups.noServices')}</div>
                            ) : (
                                services.map(svc => (
                                    <div key={svc.ID} className="flex items-center space-x-2">
                                        <Switch
                                            checked={formData.service_ids.includes(svc.ID)}
                                            onCheckedChange={() => toggleService(svc.ID)}
                                        />
                                        <span>{svc.display_name || svc.name}</span>
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
    const [groups, setGroups] = useState<Group[]>([]);
    const [services, setServices] = useState<MCPService[]>([]);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [editingGroup, setEditingGroup] = useState<Group | null>(null);

    const fetchData = async () => {
        try {
            const [groupsResp, servicesResp] = await Promise.all([
                GroupService.getAll(),
                api.get('/mcp_services/installed') // Assuming this endpoint exists based on backend files
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
        return `${baseUrl}/group/${name}/mcp?key=<YOUR_TOKEN>`;
    };

    const copyToClipboard = async (text: string) => {
        const success = await writeToClipboard(text);
        if (success) {
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

    return (
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
                    let serviceCount = 0;
                    try {
                        serviceCount = JSON.parse(group.service_ids_json || '[]').length;
                    } catch {}

                    return (
                        <Card key={group.id} className="flex flex-col">
                            <CardHeader>
                                <div className="flex justify-between items-start">
                                    <div className="space-y-1">
                                        <CardTitle className="text-xl">{group.display_name}</CardTitle>
                                        <CardDescription className="font-mono text-xs">{group.name}</CardDescription>
                                    </div>
                                    <Badge variant={group.enabled ? "default" : "secondary"}>
                                        {group.enabled ? t('common.enabled') : t('common.disabled')}
                                    </Badge>
                                </div>
                            </CardHeader>
                            <CardContent className="flex-1 space-y-4">
                                <p className="text-sm text-muted-foreground line-clamp-2 min-h-[2.5rem]">
                                    {group.description || t('groups.description')}
                                </p>
                                
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
                                            onClick={() => copyToClipboard(url)}
                                        >
                                            <Copy className="h-4 w-4" />
                                        </Button>
                                    </div>
                                </div>
                            </CardContent>
                            <div className="p-6 pt-0 mt-auto flex justify-end gap-2">
                                <Button variant="outline" size="sm" onClick={() => handleEdit(group)}>
                                    <Edit2 className="mr-2 h-3 w-3" />
                                    {t('common.edit')}
                                </Button>
                                <Button variant="destructive" size="sm" onClick={() => handleDelete(group)}>
                                    <Trash2 className="mr-2 h-3 w-3" />
                                    {t('common.delete')}
                                </Button>
                            </div>
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
    );
};

export default GroupPage;
