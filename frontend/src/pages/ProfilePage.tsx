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
import { useTranslation } from 'react-i18next';

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
    const { t } = useTranslation();

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
            } catch {
                toast({
                    variant: "destructive",
                    title: t('profile.messages.fetchUserInfoFailed'),
                    description: t('profile.messages.fetchUserInfoFailed')
                });
            } finally {
                setLoading(false);
            }
        };

        fetchUserInfo();
    }, [t, toast]);

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
                title: t('common.error'),
                description: t('profile.messages.passwordMismatch')
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
                    title: t('common.success'),
                    description: t('profile.messages.passwordChangeSuccess')
                });
                setFormData(prev => ({
                    ...prev,
                    currentPassword: '',
                    newPassword: '',
                    confirmPassword: ''
                }));
            } else {
                // 处理后端返回的错误信息
                let errorMessage = t('profile.messages.passwordChangeFailed');
                if (response.message === "current_password_incorrect") {
                    errorMessage = t('profile.messages.currentPasswordIncorrect');
                } else if (response.message === "oauth_user_cannot_change_password") {
                    errorMessage = t('profile.messages.oauthUserCannotChangePassword');
                } else if (response.message) {
                    errorMessage = response.message;
                }

                toast({
                    variant: "destructive",
                    title: t('profile.messages.passwordChangeFailed'),
                    description: errorMessage
                });
            }
        } catch (error) {
            console.error('Password change error:', error);
            toast({
                variant: "destructive",
                title: t('common.error'),
                description: t('profile.messages.passwordChangeFailed')
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
                    title: t('profile.messages.apiKeyRefreshSuccess'),
                    description: t('profile.messages.apiKeyRefreshSuccess')
                });
            }
        } catch {
            toast({
                variant: "destructive",
                title: t('profile.messages.apiKeyRefreshFailed'),
                description: t('profile.messages.apiKeyRefreshFailed')
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
                    title: t('profile.messages.profileUpdateSuccess'),
                    description: t('profile.messages.profileUpdateSuccess')
                });
            }
        } catch {
            toast({
                variant: "destructive",
                title: t('profile.messages.profileUpdateFailed'),
                description: t('profile.messages.profileUpdateFailed')
            });
        } finally {
            setSaving(false);
        }
    };

    if (loading) {
        return (
            <div className="w-full space-y-8">
                <h2 className="text-3xl font-bold tracking-tight mb-8">{t('profile.title')}</h2>
                <div className="flex justify-center items-center h-32">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
            </div>
        );
    }

    return (
        <div className="w-full space-y-8">
            <div className="flex items-center justify-between">
                <h2 className="text-3xl font-bold tracking-tight">{t('profile.title')}</h2>
                <div className="flex items-center gap-2">
                    {loginMethod === 'github' && (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <Github className="h-3 w-3" />
                            GitHub {t('auth.login')}
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
                            Google {t('auth.login')}
                        </Badge>
                    )}
                    {loginMethod === 'wechat' && (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <svg className="h-3 w-3" viewBox="0 0 24 24" fill="currentColor">
                                <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.145 4.203 2.939 5.676.135.111.239.252.287.408l.213 1.071c.033.162.145.295.287.408.142.113.317.162.489.162.172 0 .347-.049.489-.162.142-.113.254-.246.287-.408l.213-1.071c.048-.156.152-.297.287-.408C10.855 13.733 12 11.742 12 9.53c0-4.054-3.891-7.342-8.691-7.342zm-.356 3.515c.213 0 .427.016.641.049.428.066.856.165 1.284.297.428.132.814.297 1.145.489.331.192.612.408.856.652.244.244.428.508.570.814.142.306.213.652.213 1.018 0 .366-.071.712-.213 1.018-.142.306-.326.57-.57.814-.244.244-.525.46-.856.652-.331.192-.717.357-1.145.489-.428.132-.856.231-1.284.297-.214.033-.428.049-.641.049s-.427-.016-.641-.049c-.428-.066-.856-.165-1.284-.297-.428-.132-.814-.297-1.145-.489-.331-.192-.612-.408-.856-.652-.244-.244-.428-.508-.57-.814-.142-.306-.213-.652-.213-1.018 0-.366.071-.712.213-1.018.142-.306.326-.57.57-.814.244-.244.525-.46.856-.652.331-.192.717-.357 1.145-.489.428-.132.856-.231 1.284-.297.214-.033.428-.049.641-.049z" />
                            </svg>
                            {t('profile.loginMethods.wechat')}{t('auth.login')}
                        </Badge>
                    )}
                    {loginMethod === 'password' && (
                        <Badge variant="secondary" className="flex items-center gap-1">
                            <User className="h-3 w-3" />
                            {t('profile.loginMethods.password')}{t('auth.login')}
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
                                {t('profile.personalInfo')}
                            </CardTitle>
                            <CardDescription>
                                {isOAuthUser
                                    ? t('profile.notes.oauthReadOnlyInfo')
                                    : t('profile.personalInfoDesc')
                                }
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div className="space-y-2">
                                <Label htmlFor="username">{t('profile.form.username')}</Label>
                                <Input
                                    id="username"
                                    value={formData.username}
                                    onChange={(e) => setFormData(prev => ({ ...prev, username: e.target.value }))}
                                    disabled={isOAuthUser}
                                    className={isOAuthUser ? "bg-muted" : ""}
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="email">{t('profile.form.email')}</Label>
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
                                <Label htmlFor="displayName">{t('profile.form.displayName')}</Label>
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
                                        {saving ? t('profile.actions.saving') : t('profile.actions.saveProfile')}
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
                                        {t('common.cancel')}
                                    </Button>
                                </div>
                            )}

                            {isOAuthUser && (
                                <div className="pt-4 text-sm text-muted-foreground">
                                    {t('profile.notes.oauthPasswordNote')}
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
                            {t('profile.apiAccess')}
                        </CardTitle>
                        <CardDescription>
                            {t('profile.apiAccessDesc')}
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="space-y-2">
                            <Label htmlFor="apikey">{t('profile.form.apiKey')}</Label>
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
                                {refreshingToken ? t('common.loading') : t('profile.actions.refreshApiKey')}
                            </Button>
                        </div>

                        <div className="text-sm text-muted-foreground bg-muted/50 p-3 rounded-md">
                            <p className="font-medium mb-1">{t('profile.notes.apiKeyTitle')}：</p>
                            <ul className="space-y-1 text-xs">
                                <li>• {t('profile.notes.apiKeyNote1')}</li>
                                <li>• {t('profile.notes.apiKeyNote2')}</li>
                                <li>• {t('profile.notes.apiKeyNote3')}</li>
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
                            {t('profile.actions.changePassword')}
                        </CardTitle>
                        <CardDescription>
                            {t('profile.accountSecurityDesc')}
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="max-w-md space-y-4">
                            <div className="space-y-2">
                                <Label htmlFor="currentPassword">{t('profile.form.currentPassword')}</Label>
                                <Input
                                    id="currentPassword"
                                    type="password"
                                    value={formData.currentPassword}
                                    onChange={(e) => setFormData(prev => ({ ...prev, currentPassword: e.target.value }))}
                                    placeholder={t('profile.form.currentPassword')}
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="newPassword">{t('profile.form.newPassword')}</Label>
                                <Input
                                    id="newPassword"
                                    type="password"
                                    value={formData.newPassword}
                                    onChange={(e) => setFormData(prev => ({ ...prev, newPassword: e.target.value }))}
                                    placeholder={t('profile.form.newPassword')}
                                />
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="confirmPassword">{t('profile.form.confirmPassword')}</Label>
                                <Input
                                    id="confirmPassword"
                                    type="password"
                                    value={formData.confirmPassword}
                                    onChange={(e) => setFormData(prev => ({ ...prev, confirmPassword: e.target.value }))}
                                    placeholder={t('profile.form.confirmPassword')}
                                />
                            </div>
                        </div>

                        <Button
                            onClick={handleChangePassword}
                            disabled={!formData.currentPassword || !formData.newPassword || !formData.confirmPassword || saving}
                        >
                            {saving ? t('common.loading') : t('profile.actions.changePassword')}
                        </Button>
                    </CardContent>
                </Card>
            )}
        </div>
    );
} 