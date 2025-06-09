import { useState, useEffect } from 'react';
import { useOutletContext } from 'react-router-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { useToast } from '@/hooks/use-toast';
// import { useAuth } from '@/contexts/AuthContext'; // 暂时未使用
import { Eye, EyeOff, RefreshCw, Github, User, Lock } from 'lucide-react';
import api, { APIResponse } from '@/utils/api';
import type { PageOutletContext } from '../App';

interface UserInfo {
    id: number;
    username: string;
    email: string;
    display_name: string;
    role: number;
    status: number;
    token: string;
    github_id?: string;
    google_id?: string;
    wechat_id?: string;
}

export function ProfilePage() {
    useOutletContext<PageOutletContext>();
    // const { currentUser } = useAuth(); // 暂时未使用
    const { toast } = useToast();

    const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
    const [loading, setLoading] = useState(true);
    const [showApiKey, setShowApiKey] = useState(false);
    // const [editMode, setEditMode] = useState(true); // 不再需要编辑模式
    const [saving, setSaving] = useState(false);
    const [refreshingToken, setRefreshingToken] = useState(false);

    // 表单数据
    const [formData, setFormData] = useState({
        username: '',
        email: '',
        displayName: '',
        currentPassword: '',
        newPassword: '',
        confirmPassword: ''
    });

    // 获取用户详细信息
    useEffect(() => {
        const fetchUserInfo = async () => {
            try {
                const response: APIResponse = await api.get('/user/self');
                if (response.success && response.data) {
                    setUserInfo(response.data);
                    setFormData({
                        username: response.data.username || '',
                        email: response.data.email || '',
                        displayName: response.data.display_name || '',
                        currentPassword: '',
                        newPassword: '',
                        confirmPassword: ''
                    });
                }
            } catch (error) {
                toast({
                    variant: "destructive",
                    title: "获取用户信息失败",
                    description: "请稍后重试"
                });
            } finally {
                setLoading(false);
            }
        };

        fetchUserInfo();
    }, [toast]);

    // 判断登录方式
    const getLoginMethod = () => {
        if (!userInfo) return 'password';
        if (userInfo.github_id) return 'github';
        if (userInfo.google_id) return 'google';
        if (userInfo.wechat_id) return 'wechat';
        return 'password';
    };

    const loginMethod = getLoginMethod();
    const isOAuthUser = loginMethod !== 'password';

    // 格式化显示的 API Key
    const formatApiKey = (token: string) => {
        if (!token) return '';
        if (showApiKey) return token;
        return token.substring(0, 8) + '••••••••••••••••' + token.substring(token.length - 4);
    };

    // 修改密码
    const handleChangePassword = async () => {
        if (formData.newPassword !== formData.confirmPassword) {
            toast({
                variant: "destructive",
                title: "错误",
                description: "新密码和确认密码不匹配"
            });
            return;
        }

        setSaving(true);
        try {
            const response: APIResponse = await api.post('/user/change-password', {
                current_password: formData.currentPassword,
                new_password: formData.newPassword
            });

            if (response.success) {
                toast({
                    title: "成功",
                    description: "密码修改成功"
                });
                setFormData(prev => ({
                    ...prev,
                    currentPassword: '',
                    newPassword: '',
                    confirmPassword: ''
                }));
            } else {
                // 处理后端返回的错误信息
                let errorMessage = "密码修改失败";
                if (response.message === "current_password_incorrect") {
                    errorMessage = "当前密码不正确，请重新输入";
                } else if (response.message === "oauth_user_cannot_change_password") {
                    errorMessage = "OAuth 用户无法修改密码";
                } else if (response.message) {
                    errorMessage = response.message;
                }

                toast({
                    variant: "destructive",
                    title: "密码修改失败",
                    description: errorMessage
                });
            }
        } catch (error) {
            console.error('Password change error:', error);
            toast({
                variant: "destructive",
                title: "错误",
                description: "网络错误，请稍后重试"
            });
        } finally {
            setSaving(false);
        }
    };

    // 刷新 API Key
    const handleRefreshApiKey = async () => {
        setRefreshingToken(true);
        try {
            const response: APIResponse = await api.get('/user/token');
            if (response.success && response.data) {
                setUserInfo(prev => prev ? { ...prev, token: response.data } : null);
                toast({
                    title: "API Key 已更新",
                    description: "新的 API Key 已生成"
                });
            }
        } catch (error) {
            toast({
                variant: "destructive",
                title: "更新失败",
                description: "API Key 更新失败，请稍后重试"
            });
        } finally {
            setRefreshingToken(false);
        }
    };

    // 保存个人信息
    const handleSaveProfile = async () => {
        setSaving(true);
        try {
            const updateData: any = {
                username: formData.username,
                email: formData.email,
                display_name: formData.displayName
            };

            const response: APIResponse = await api.put('/user/self', updateData);
            if (response.success) {
                setUserInfo(prev => prev ? {
                    ...prev,
                    username: formData.username,
                    email: formData.email,
                    display_name: formData.displayName
                } : null);
                toast({
                    title: "更新成功",
                    description: "个人信息已更新"
                });
            }
        } catch (error) {
            toast({
                variant: "destructive",
                title: "更新失败",
                description: "个人信息更新失败，请稍后重试"
            });
        } finally {
            setSaving(false);
        }
    };

    if (loading) {
        return (
            <div className="w-full space-y-8">
                <h2 className="text-3xl font-bold tracking-tight mb-8">用户资料</h2>
                <div className="flex justify-center items-center h-32">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
            </div>
        );
    }

    return (
        <div className="w-full space-y-8">
            <div className="flex items-center justify-between">
                <h2 className="text-3xl font-bold tracking-tight">用户资料</h2>
                <div className="flex items-center gap-2">
                    {loginMethod === 'github' && (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <Github className="h-3 w-3" />
                            GitHub 登录
                        </Badge>
                    )}
                    {loginMethod === 'google' && (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <svg className="h-3 w-3" viewBox="0 0 24 24" fill="currentColor">
                                <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4" />
                                <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
                                <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
                                <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
                            </svg>
                            Google 登录
                        </Badge>
                    )}
                    {loginMethod === 'wechat' && (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <svg className="h-3 w-3" viewBox="0 0 24 24" fill="currentColor">
                                <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.145 4.203 2.939 5.676.135.111.239.252.287.408l.213 1.071c.033.162.145.295.287.408.142.113.317.162.489.162.172 0 .347-.049.489-.162.142-.113.254-.246.287-.408l.213-1.071c.048-.156.152-.297.287-.408C10.855 13.733 12 11.742 12 9.53c0-4.054-3.891-7.342-8.691-7.342zm-.356 3.515c.213 0 .427.016.641.049.428.066.856.165 1.284.297.428.132.814.297 1.145.489.331.192.612.408.856.652.244.244.428.508.570.814.142.306.213.652.213 1.018 0 .366-.071.712-.213 1.018-.142.306-.326.57-.57.814-.244.244-.525.46-.856.652-.331.192-.717.357-1.145.489-.428.132-.856.231-1.284.297-.214.033-.428.049-.641.049s-.427-.016-.641-.049c-.428-.066-.856-.165-1.284-.297-.428-.132-.814-.297-1.145-.489-.331-.192-.612-.408-.856-.652-.244-.244-.428-.508-.57-.814-.142-.306-.213-.652-.213-1.018 0-.366.071-.712.213-1.018.142-.306.326-.57.57-.814.244-.244.525-.46.856-.652.331-.192.717-.357 1.145-.489.428-.132.856-.231 1.284-.297.214-.033.428-.049.641-.049z" />
                            </svg>
                            微信登录
                        </Badge>
                    )}
                    {loginMethod === 'password' && (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <User className="h-3 w-3" />
                            账号密码登录
                        </Badge>
                    )}
                </div>
            </div>

            <div className="grid gap-6 md:grid-cols-1 lg:grid-cols-2">
                {/* 个人信息卡片 - OAuth 用户隐藏 */}
                {!isOAuthUser && (
                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <User className="h-5 w-5" />
                                个人信息
                            </CardTitle>
                            <CardDescription>
                                {isOAuthUser
                                    ? "通过第三方登录的用户信息（只读）"
                                    : "管理您的个人账户信息"
                                }
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div className="space-y-2">
                                <Label htmlFor="username">用户名</Label>
                                <Input
                                    id="username"
                                    value={formData.username}
                                    onChange={(e) => setFormData(prev => ({ ...prev, username: e.target.value }))}
                                    disabled={isOAuthUser}
                                    className={isOAuthUser ? "bg-muted" : ""}
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="email">邮箱地址</Label>
                                <Input
                                    id="email"
                                    type="email"
                                    value={formData.email}
                                    onChange={(e) => setFormData(prev => ({ ...prev, email: e.target.value }))}
                                    disabled={isOAuthUser}
                                    className={isOAuthUser ? "bg-muted" : ""}
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="displayName">显示名称</Label>
                                <Input
                                    id="displayName"
                                    value={formData.displayName}
                                    onChange={(e) => setFormData(prev => ({ ...prev, displayName: e.target.value }))}
                                    disabled={isOAuthUser}
                                    className={isOAuthUser ? "bg-muted" : ""}
                                />
                            </div>

                            {!isOAuthUser && (
                                <div className="flex gap-2 pt-4">
                                    <Button onClick={handleSaveProfile} disabled={saving}>
                                        {saving ? "保存中..." : "保存更改"}
                                    </Button>
                                    <Button
                                        variant="outline"
                                        onClick={() => {
                                            setFormData({
                                                username: userInfo?.username || '',
                                                email: userInfo?.email || '',
                                                displayName: userInfo?.display_name || '',
                                                currentPassword: '',
                                                newPassword: '',
                                                confirmPassword: ''
                                            });
                                        }}
                                    >
                                        重置
                                    </Button>
                                </div>
                            )}

                            {isOAuthUser && (
                                <div className="pt-4 text-sm text-muted-foreground">
                                    通过第三方登录的用户无法修改个人信息
                                </div>
                            )}
                        </CardContent>
                    </Card>
                )}

                {/* API Key 管理卡片 */}
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                            <Lock className="h-5 w-5" />
                            API Key 管理
                        </CardTitle>
                        <CardDescription>
                            用于 API 请求的身份验证密钥
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="space-y-2">
                            <Label htmlFor="apikey">当前 API Key</Label>
                            <div className="flex gap-2">
                                <Input
                                    id="apikey"
                                    value={formatApiKey(userInfo?.token || '')}
                                    disabled
                                    className="bg-muted font-mono text-sm"
                                />
                                <Button
                                    variant="outline"
                                    size="icon"
                                    onClick={() => setShowApiKey(!showApiKey)}
                                >
                                    {showApiKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                                </Button>
                            </div>
                        </div>

                        <div className="space-y-2">
                            <Button
                                onClick={handleRefreshApiKey}
                                disabled={refreshingToken}
                                className="w-full"
                            >
                                <RefreshCw className={`h-4 w-4 mr-2 ${refreshingToken ? 'animate-spin' : ''}`} />
                                {refreshingToken ? "生成中..." : "重新生成 API Key"}
                            </Button>
                        </div>

                        <div className="text-sm text-muted-foreground bg-muted/50 p-3 rounded-md">
                            <p className="font-medium mb-1">注意事项：</p>
                            <ul className="space-y-1 text-xs">
                                <li>• 重新生成后，旧的 API Key 将立即失效</li>
                                <li>• 请及时更新您的应用程序配置</li>
                                <li>• 请妥善保管您的 API Key</li>
                            </ul>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* 密码修改卡片（仅限账号密码登录用户） */}
            {!isOAuthUser && (
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                            <Lock className="h-5 w-5" />
                            修改密码
                        </CardTitle>
                        <CardDescription>
                            更新您的登录密码
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="max-w-md space-y-4">
                            <div className="space-y-2">
                                <Label htmlFor="currentPassword">当前密码</Label>
                                <Input
                                    id="currentPassword"
                                    type="password"
                                    value={formData.currentPassword}
                                    onChange={(e) => setFormData(prev => ({ ...prev, currentPassword: e.target.value }))}
                                    placeholder="请输入当前密码"
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="newPassword">新密码</Label>
                                <Input
                                    id="newPassword"
                                    type="password"
                                    value={formData.newPassword}
                                    onChange={(e) => setFormData(prev => ({ ...prev, newPassword: e.target.value }))}
                                    placeholder="请输入新密码"
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="confirmPassword">确认新密码</Label>
                                <Input
                                    id="confirmPassword"
                                    type="password"
                                    value={formData.confirmPassword}
                                    onChange={(e) => setFormData(prev => ({ ...prev, confirmPassword: e.target.value }))}
                                    placeholder="请再次输入新密码"
                                />
                            </div>
                        </div>

                        <Button
                            onClick={handleChangePassword}
                            disabled={!formData.currentPassword || !formData.newPassword || !formData.confirmPassword || saving}
                        >
                            {saving ? "修改中..." : "修改密码"}
                        </Button>
                    </CardContent>
                </Card>
            )}
        </div>
    );
} 