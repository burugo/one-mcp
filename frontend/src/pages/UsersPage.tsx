import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableHead, TableHeader, TableRow, TableCell } from '@/components/ui/table';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { useToast } from '@/hooks/use-toast';
import { Search, Plus, Edit, Trash2, UserCheck } from 'lucide-react';
import api, { APIResponse } from '@/utils/api';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { UserDialog } from '@/components/users/UserDialog';
import { useTranslation } from 'react-i18next';

interface User {
    id: number;
    username: string;
    display_name: string;
    email: string;
    role: number;
    status: number;
    github_id?: string;
    google_id?: string;
    wechat_id?: string;
    created_at: string;
    updated_at: string;
}

export function UsersPage() {
    const { t } = useTranslation();
    const { toast } = useToast();
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const [currentPage, setCurrentPage] = useState(0);
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
    const [pendingDeleteId, setPendingDeleteId] = useState<number | null>(null);
    const [userDialogOpen, setUserDialogOpen] = useState(false);
    const [currentUserForEdit, setCurrentUserForEdit] = useState<User | null>(null);

    // 获取用户列表
    const fetchUsers = async (page = 0, search = '') => {
        setLoading(true);
        try {
            let url = `/user/?p=${page}`;
            if (search.trim()) {
                url = `/user/search?keyword=${encodeURIComponent(search.trim())}`;
            }

            const response = await api.get(url) as APIResponse<User[]>;
            if (response.success) {
                setUsers(response.data || []);
            } else {
                toast({
                    title: '获取用户列表失败',
                    description: response.message || '未知错误',
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: '获取用户列表失败',
                description: error.message || '网络错误',
                variant: 'destructive'
            });
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchUsers(currentPage, searchTerm);
    }, [currentPage]);

    // 搜索处理
    const handleSearch = () => {
        setCurrentPage(0);
        fetchUsers(0, searchTerm);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            handleSearch();
        }
    };

    // 删除用户
    const handleDeleteClick = (userId: number) => {
        setPendingDeleteId(userId);
        setDeleteDialogOpen(true);
    };

    const handleDeleteConfirm = async () => {
        if (!pendingDeleteId) return;

        try {
            const response = await api.delete(`/user/${pendingDeleteId}`) as APIResponse<any>;
            if (response.success) {
                toast({
                    title: t('usersPage.messages.deleteSuccess'),
                    description: t('usersPage.messages.userDeleted')
                });
                fetchUsers(currentPage, searchTerm);
            } else {
                toast({
                    title: t('usersPage.messages.deleteFailed'),
                    description: response.message || t('usersPage.messages.unknownError'),
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: t('usersPage.messages.deleteFailed'),
                description: error.message || t('usersPage.messages.networkError'),
                variant: 'destructive'
            });
        } finally {
            setDeleteDialogOpen(false);
            setPendingDeleteId(null);
        }
    };

    // 设为管理员
    const handlePromoteToAdmin = async (username: string) => {
        try {
            const response = await api.post('/user/manage', {
                username,
                action: 'promote'
            }) as APIResponse<any>;

            if (response.success) {
                toast({
                    title: t('usersPage.messages.operationSuccess'),
                    description: t('usersPage.messages.userPromoted')
                });
                fetchUsers(currentPage, searchTerm);
            } else {
                toast({
                    title: t('usersPage.messages.operationFailed'),
                    description: response.message || t('usersPage.messages.unknownError'),
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: t('usersPage.messages.operationFailed'),
                description: error.message || t('usersPage.messages.networkError'),
                variant: 'destructive'
            });
        }
    };

    // 设为普通用户
    const handleDemoteToUser = async (username: string) => {
        try {
            const response = await api.post('/user/manage', {
                username,
                action: 'demote'
            }) as APIResponse<any>;

            if (response.success) {
                toast({
                    title: t('usersPage.messages.operationSuccess'),
                    description: t('usersPage.messages.userDemoted')
                });
                fetchUsers(currentPage, searchTerm);
            } else {
                toast({
                    title: t('usersPage.messages.operationFailed'),
                    description: response.message || t('usersPage.messages.unknownError'),
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: t('usersPage.messages.operationFailed'),
                description: error.message || t('usersPage.messages.networkError'),
                variant: 'destructive'
            });
        }
    };

    // 切换用户状态
    const handleToggleStatus = async (username: string, currentStatus: number) => {
        const action = currentStatus === 1 ? 'disable' : 'enable';
        try {
            const response = await api.post('/user/manage', {
                username,
                action
            }) as APIResponse<any>;

            if (response.success) {
                toast({
                    title: t('usersPage.messages.operationSuccess'),
                    description: action === 'enable' ? t('usersPage.messages.userEnabled') : t('usersPage.messages.userDisabled')
                });
                fetchUsers(currentPage, searchTerm);
            } else {
                toast({
                    title: t('usersPage.messages.operationFailed'),
                    description: response.message || t('usersPage.messages.unknownError'),
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: t('usersPage.messages.operationFailed'),
                description: error.message || t('usersPage.messages.networkError'),
                variant: 'destructive'
            });
        }
    };

    // 获取角色显示文本
    const getRoleText = (role: number) => {
        switch (role) {
            case 100: return t('usersPage.roles.superAdmin');
            case 10: return t('usersPage.roles.admin');
            case 1: return t('usersPage.roles.user');
            default: return t('usersPage.roles.unknown');
        }
    };

    // 获取绑定状态
    const getBindingStatus = (user: User) => {
        const bindings = [];
        if (user.github_id) bindings.push('GitHub');
        if (user.google_id) bindings.push('Google');
        if (user.wechat_id) bindings.push('WeChat');
        return bindings.length > 0 ? bindings.join(', ') : t('usersPage.bindings.none');
    };

    const handleOpenNewUserDialog = () => {
        setCurrentUserForEdit(null);
        setUserDialogOpen(true);
    };

    const handleOpenEditUserDialog = (user: User) => {
        setCurrentUserForEdit(user);
        setUserDialogOpen(true);
    };

    const handleUserDialogClose = () => {
        setUserDialogOpen(false);
        setCurrentUserForEdit(null);
    };

    const handleUserSaved = () => {
        fetchUsers(currentPage, searchTerm);
    };

    return (
        <div className="w-full space-y-6">
            <div className="flex justify-between items-center">
                <div>
                    <h2 className="text-3xl font-bold tracking-tight">{t('usersPage.title')}</h2>
                    <p className="text-muted-foreground mt-1">{t('usersPage.description')}</p>
                </div>
                <Button className="bg-[#7c3aed] hover:bg-[#7c3aed]/90" onClick={handleOpenNewUserDialog}>
                    <Plus className="w-4 h-4 mr-2" />
                    {t('usersPage.addUser')}
                </Button>
            </div>

            {/* 搜索框 */}
            <div className="flex gap-4 mb-6">
                <div className="relative flex-grow">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                        className="pl-10 bg-muted/40"
                        placeholder={t('usersPage.searchPlaceholder')}
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        onKeyDown={handleKeyDown}
                    />
                </div>
                <Button onClick={handleSearch} disabled={loading}>
                    {loading ? t('usersPage.searching') : t('usersPage.search')}
                </Button>
            </div>

            {/* 用户列表表格 */}
            <div className="border rounded-lg">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{t('usersPage.id')}</TableHead>
                            <TableHead>{t('usersPage.username')}</TableHead>
                            <TableHead>{t('usersPage.displayName')}</TableHead>
                            <TableHead>{t('usersPage.email')}</TableHead>
                            <TableHead>{t('usersPage.userRole')}</TableHead>
                            <TableHead>{t('usersPage.binding')}</TableHead>
                            <TableHead>{t('usersPage.status')}</TableHead>
                            <TableHead>{t('usersPage.actions')}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={8} className="text-center py-8">
                                    {t('usersPage.loading')}
                                </TableCell>
                            </TableRow>
                        ) : users.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                                    {t('usersPage.noUsersFound')}
                                </TableCell>
                            </TableRow>
                        ) : (
                            users.map((user) => (
                                <TableRow key={user.id}>
                                    <TableCell className="font-medium">{user.id}</TableCell>
                                    <TableCell>{user.username}</TableCell>
                                    <TableCell>{user.display_name}</TableCell>
                                    <TableCell>{user.email || '-'}</TableCell>
                                    <TableCell>
                                        <Badge variant={user.role >= 10 ? 'default' : 'secondary'}>
                                            {getRoleText(user.role)}
                                        </Badge>
                                    </TableCell>
                                    <TableCell>
                                        <span className="text-sm text-muted-foreground">
                                            {getBindingStatus(user)}
                                        </span>
                                    </TableCell>
                                    <TableCell>
                                        <Switch
                                            checked={user.status === 1}
                                            onCheckedChange={() => handleToggleStatus(user.username, user.status)}
                                        />
                                    </TableCell>
                                    <TableCell>
                                        <div className="flex items-center space-x-2">
                                            {/* 角色切换按钮 - 根据当前角色显示不同操作 */}
                                            {user.role < 10 ? (
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handlePromoteToAdmin(user.username)}
                                                    title={t('usersPage.promoteToAdmin')}
                                                >
                                                    <UserCheck className="w-4 h-4" />
                                                </Button>
                                            ) : user.role === 10 ? (
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleDemoteToUser(user.username)}
                                                    title={t('usersPage.demoteToUser')}
                                                    className="text-orange-500 hover:text-orange-700"
                                                >
                                                    <UserCheck className="w-4 h-4" />
                                                </Button>
                                            ) : null}
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                onClick={() => handleOpenEditUserDialog(user)}
                                                title={t('usersPage.edit')}
                                            >
                                                <Edit className="w-4 h-4" />
                                            </Button>
                                            {user.role !== 100 && (
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleDeleteClick(user.id)}
                                                    title={t('usersPage.delete')}
                                                    className="text-red-500 hover:text-red-700"
                                                >
                                                    <Trash2 className="w-4 h-4" />
                                                </Button>
                                            )}
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            {/* 分页控件 */}
            <div className="flex justify-between items-center">
                <div className="text-sm text-muted-foreground">
                    {t('usersPage.showing')} {users.length} {t('usersPage.users')}
                </div>
                <div className="flex space-x-2">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage(Math.max(0, currentPage - 1))}
                        disabled={currentPage === 0 || loading}
                    >
                        {t('usersPage.previousPage')}
                    </Button>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage(currentPage + 1)}
                        disabled={users.length < 10 || loading}
                    >
                        {t('usersPage.nextPage')}
                    </Button>
                </div>
            </div>

            {/* 删除确认对话框 */}
            <ConfirmDialog
                isOpen={deleteDialogOpen}
                onOpenChange={setDeleteDialogOpen}
                title={t('usersPage.confirmDeleteTitle')}
                description={t('usersPage.confirmDeleteDescription')}
                confirmText={t('usersPage.delete')}
                cancelText={t('usersPage.cancel')}
                onConfirm={handleDeleteConfirm}
                confirmButtonVariant="destructive"
            />

            {/* 用户新增/编辑对话框 */}
            <UserDialog
                isOpen={userDialogOpen}
                onClose={handleUserDialogClose}
                onSave={handleUserSaved}
                currentUser={currentUserForEdit}
            />
        </div>
    );
} 