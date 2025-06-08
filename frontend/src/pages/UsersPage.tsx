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
                    title: '删除成功',
                    description: '用户已成功删除'
                });
                fetchUsers(currentPage, searchTerm);
            } else {
                toast({
                    title: '删除失败',
                    description: response.message || '未知错误',
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: '删除失败',
                description: error.message || '网络错误',
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
                    title: '操作成功',
                    description: '用户已设为管理员'
                });
                fetchUsers(currentPage, searchTerm);
            } else {
                toast({
                    title: '操作失败',
                    description: response.message || '未知错误',
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: '操作失败',
                description: error.message || '网络错误',
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
                    title: '操作成功',
                    description: '用户已设为普通用户'
                });
                fetchUsers(currentPage, searchTerm);
            } else {
                toast({
                    title: '操作失败',
                    description: response.message || '未知错误',
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: '操作失败',
                description: error.message || '网络错误',
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
                    title: '操作成功',
                    description: `用户已${action === 'enable' ? '启用' : '禁用'}`
                });
                fetchUsers(currentPage, searchTerm);
            } else {
                toast({
                    title: '操作失败',
                    description: response.message || '未知错误',
                    variant: 'destructive'
                });
            }
        } catch (error: any) {
            toast({
                title: '操作失败',
                description: error.message || '网络错误',
                variant: 'destructive'
            });
        }
    };

    // 获取角色显示文本
    const getRoleText = (role: number) => {
        switch (role) {
            case 100: return '超级管理员';
            case 10: return '管理员';
            case 1: return '普通用户';
            default: return '未知';
        }
    };

    // 获取绑定状态
    const getBindingStatus = (user: User) => {
        const bindings = [];
        if (user.github_id) bindings.push('GitHub');
        if (user.google_id) bindings.push('Google');
        if (user.wechat_id) bindings.push('WeChat');
        return bindings.length > 0 ? bindings.join(', ') : '无';
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
                    <h2 className="text-3xl font-bold tracking-tight">用户管理</h2>
                    <p className="text-muted-foreground mt-1">管理系统用户账户</p>
                </div>
                <Button className="bg-[#7c3aed] hover:bg-[#7c3aed]/90" onClick={handleOpenNewUserDialog}>
                    <Plus className="w-4 h-4 mr-2" />
                    新增用户
                </Button>
            </div>

            {/* 搜索框 */}
            <div className="flex gap-4 mb-6">
                <div className="relative flex-grow">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                        className="pl-10 bg-muted/40"
                        placeholder="搜索用户ID、用户名、显示名称或邮箱..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        onKeyDown={handleKeyDown}
                    />
                </div>
                <Button onClick={handleSearch} disabled={loading}>
                    {loading ? '搜索中...' : '搜索'}
                </Button>
            </div>

            {/* 用户列表表格 */}
            <div className="border rounded-lg">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>ID</TableHead>
                            <TableHead>用户名</TableHead>
                            <TableHead>显示名称</TableHead>
                            <TableHead>邮箱</TableHead>
                            <TableHead>用户角色</TableHead>
                            <TableHead>绑定</TableHead>
                            <TableHead>状态</TableHead>
                            <TableHead>操作</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading ? (
                            <TableRow>
                                <TableCell colSpan={8} className="text-center py-8">
                                    加载中...
                                </TableCell>
                            </TableRow>
                        ) : users.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                                    没有找到用户
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
                                                    title="设为管理员"
                                                >
                                                    <UserCheck className="w-4 h-4" />
                                                </Button>
                                            ) : user.role === 10 ? (
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleDemoteToUser(user.username)}
                                                    title="设为普通用户"
                                                    className="text-orange-500 hover:text-orange-700"
                                                >
                                                    <UserCheck className="w-4 h-4" />
                                                </Button>
                                            ) : null}
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                onClick={() => handleOpenEditUserDialog(user)}
                                                title="编辑"
                                            >
                                                <Edit className="w-4 h-4" />
                                            </Button>
                                            {user.role !== 100 && (
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleDeleteClick(user.id)}
                                                    title="删除"
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
                    显示 {users.length} 个用户
                </div>
                <div className="flex space-x-2">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage(Math.max(0, currentPage - 1))}
                        disabled={currentPage === 0 || loading}
                    >
                        上一页
                    </Button>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage(currentPage + 1)}
                        disabled={users.length < 10 || loading}
                    >
                        下一页
                    </Button>
                </div>
            </div>

            {/* 删除确认对话框 */}
            <ConfirmDialog
                isOpen={deleteDialogOpen}
                onOpenChange={setDeleteDialogOpen}
                title="确认删除"
                description="确定要删除此用户吗？此操作不可撤销。"
                confirmText="删除"
                cancelText="取消"
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