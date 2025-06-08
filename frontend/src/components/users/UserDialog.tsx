import React, { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useToast } from '@/hooks/use-toast';
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
                    title: currentUser ? '更新成功' : '创建成功',
                    description: currentUser ? '用户信息已更新' : '用户已成功创建'
                });
                onSave(); // Trigger refresh of user list
                onClose();
            } else {
                toast({
                    title: currentUser ? '更新失败' : '创建失败',
                    description: response.message || '未知错误',
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: currentUser ? '更新失败' : '创建失败',
                description: error.message || '网络错误',
                variant: 'destructive'
            });
        } finally {
            setLoading(false);
        }
    };

    const dialogTitle = currentUser ? '编辑用户' : '新增用户';
    const dialogDescription = currentUser ? '修改用户的用户名、显示名称和密码。' : '创建一个新用户帐户。'

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>{dialogTitle}</DialogTitle>
                    <DialogDescription>{dialogDescription}</DialogDescription>
                </DialogHeader>
                <div className="grid gap-4 py-4">
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="username" className="text-right">用户名</Label>
                        <Input
                            id="username"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            className="col-span-3"
                            disabled={!!currentUser} // Username typically not editable after creation
                        />
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="displayName" className="text-right">显示名称</Label>
                        <Input
                            id="displayName"
                            value={displayName}
                            onChange={(e) => setDisplayName(e.target.value)}
                            className="col-span-3"
                        />
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="password" className="text-right">密码</Label>
                        <Input
                            id="password"
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="col-span-3"
                            placeholder={currentUser ? '留空则不修改密码' : '输入密码'}
                        />
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={onClose} disabled={loading}>取消</Button>
                    <Button onClick={handleSubmit} disabled={loading}>
                        {loading ? '保存中...' : '保存'}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
} 