import React, { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useToast } from '@/hooks/use-toast';
import { useTranslation } from 'react-i18next';
import api, { APIResponse } from '@/utils/api';

interface UserDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: () => void;
    currentUser?: { // Optional, for edit mode
        id: number;
        username: string;
        display_name: string;
    } | null;
}

export const UserDialog: React.FC<UserDialogProps> = ({ isOpen, onClose, onSave, currentUser }) => {
    const { t } = useTranslation();
    const { toast } = useToast();
    const [username, setUsername] = useState('');
    const [displayName, setDisplayName] = useState('');
    const [password, setPassword] = useState('');
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (isOpen && currentUser) {
            setUsername(currentUser.username);
            setDisplayName(currentUser.display_name);
            setPassword(''); // Password is not editable directly, or set to empty for security
        } else if (isOpen) {
            // Reset form for new user
            setUsername('');
            setDisplayName('');
            setPassword('');
        }
    }, [isOpen, currentUser]);

    const handleSubmit = async () => {
        setLoading(true);
        try {
            let response: APIResponse;
            if (currentUser) {
                // Edit mode
                response = await api.put('/user', {
                    id: currentUser.id,
                    username,
                    display_name: displayName,
                    // Only send password if it's explicitly changed and not empty
                    ...(password && { password }),
                });
            } else {
                // New user mode
                response = await api.post('/user', {
                    username,
                    display_name: displayName,
                    password,
                });
            }

            if (response.success) {
                toast({
                    title: currentUser ? t('userDialog.messages.updateSuccess') : t('userDialog.messages.createSuccess'),
                    description: currentUser ? t('userDialog.messages.userUpdated') : t('userDialog.messages.userCreated')
                });
                onSave(); // Trigger refresh of user list
                onClose();
            } else {
                toast({
                    title: currentUser ? t('userDialog.messages.updateFailed') : t('userDialog.messages.createFailed'),
                    description: response.message || t('userDialog.messages.unknownError'),
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: currentUser ? t('userDialog.messages.updateFailed') : t('userDialog.messages.createFailed'),
                description: error.message || t('userDialog.messages.networkError'),
                variant: 'destructive'
            });
        } finally {
            setLoading(false);
        }
    };

    const dialogTitle = currentUser ? t('userDialog.editTitle') : t('userDialog.addTitle');
    const dialogDescription = currentUser ? t('userDialog.editDescription') : t('userDialog.addDescription');

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>{dialogTitle}</DialogTitle>
                    <DialogDescription>{dialogDescription}</DialogDescription>
                </DialogHeader>
                <div className="grid gap-4 py-4">
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="username" className="text-right">{t('userDialog.form.username')}</Label>
                        <Input
                            id="username"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            className="col-span-3"
                            disabled={!!currentUser} // Username typically not editable after creation
                        />
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="displayName" className="text-right">{t('userDialog.form.displayName')}</Label>
                        <Input
                            id="displayName"
                            value={displayName}
                            onChange={(e) => setDisplayName(e.target.value)}
                            className="col-span-3"
                        />
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="password" className="text-right">{t('userDialog.form.password')}</Label>
                        <Input
                            id="password"
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="col-span-3"
                            placeholder={currentUser ? t('userDialog.form.passwordEditPlaceholder') : t('userDialog.form.passwordPlaceholder')}
                        />
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={onClose} disabled={loading}>{t('userDialog.actions.cancel')}</Button>
                    <Button onClick={handleSubmit} disabled={loading}>
                        {loading ? t('userDialog.actions.saving') : t('userDialog.actions.save')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
} 