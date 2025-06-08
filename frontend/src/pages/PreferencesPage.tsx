import { useEffect, useState } from 'react';
import api, { APIResponse } from '@/utils/api';
import { useServerAddressStore } from '@/hooks/useServerAddress';
import { useToast } from '@/hooks/use-toast';

export function PreferencesPage() {
    const { toast } = useToast();
    const serverAddress = useServerAddressStore(s => s.serverAddress);
    const setServerAddress = useServerAddressStore(s => s.setServerAddress);
    const fetchServerAddress = useServerAddressStore(s => s.fetchServerAddress);
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [message, setMessage] = useState('');

    // OAuth configurations
    const [githubClientId, setGithubClientId] = useState('');
    const [githubClientSecret, setGithubClientSecret] = useState('');
    const [googleClientId, setGoogleClientId] = useState('');
    const [googleClientSecret, setGoogleClientSecret] = useState('');
    const [savingGithub, setSavingGithub] = useState(false);
    const [savingGoogle, setSavingGoogle] = useState(false);

    // OAuth enabled switches
    const [githubOAuthEnabled, setGithubOAuthEnabled] = useState(false);
    const [googleOAuthEnabled, setGoogleOAuthEnabled] = useState(false);
    const [savingGithubEnabled, setSavingGithubEnabled] = useState(false);
    const [savingGoogleEnabled, setSavingGoogleEnabled] = useState(false);

    useEffect(() => {
        setLoading(true);
        Promise.all([
            fetchServerAddress(),
            loadOAuthConfigs()
        ]).finally(() => setLoading(false));
    }, [fetchServerAddress]);

    const loadOAuthConfigs = async () => {
        try {
            const res = await api.get('/option/') as APIResponse;
            if (res.success && Array.isArray(res.data)) {
                const githubClientIdOption = res.data.find((item: any) => item.key === 'GitHubClientId');
                const githubClientSecretOption = res.data.find((item: any) => item.key === 'GitHubClientSecret');
                const googleClientIdOption = res.data.find((item: any) => item.key === 'GoogleClientId');
                const googleClientSecretOption = res.data.find((item: any) => item.key === 'GoogleClientSecret');
                const githubOAuthEnabledOption = res.data.find((item: any) => item.key === 'GitHubOAuthEnabled');
                const googleOAuthEnabledOption = res.data.find((item: any) => item.key === 'GoogleOAuthEnabled');

                if (githubClientIdOption) setGithubClientId(githubClientIdOption.value);
                if (githubClientSecretOption) setGithubClientSecret(githubClientSecretOption.value);
                if (googleClientIdOption) setGoogleClientId(googleClientIdOption.value);
                if (googleClientSecretOption) setGoogleClientSecret(googleClientSecretOption.value);
                if (githubOAuthEnabledOption) setGithubOAuthEnabled(githubOAuthEnabledOption.value === 'true');
                if (googleOAuthEnabledOption) setGoogleOAuthEnabled(googleOAuthEnabledOption.value === 'true');
            }
        } catch (error) {
            console.error('Failed to load OAuth configs:', error);
        }
    };

    const handleSave = async () => {
        setSaving(true);
        setMessage('');
        const clean = serverAddress.replace(/\/$/, '');
        const res = await api.put('/option/', { key: 'ServerAddress', value: clean }) as APIResponse;
        if (res.success) {
            setMessage('保存成功');
            setServerAddress(clean); // 立即同步到全局
        } else {
            setMessage(res.message || '保存失败');
        }
        setSaving(false);
    };

    const handleSaveGitHubOAuth = async () => {
        setSavingGithub(true);
        try {
            const clientIdRes = await api.put('/option/', { key: 'GitHubClientId', value: githubClientId }) as APIResponse;
            const clientSecretRes = await api.put('/option/', { key: 'GitHubClientSecret', value: githubClientSecret }) as APIResponse;

            if (clientIdRes.success && clientSecretRes.success) {
                toast({
                    title: "保存成功",
                    description: "GitHub OAuth 配置已保存"
                });
            } else {
                toast({
                    variant: "destructive",
                    title: "保存失败",
                    description: clientIdRes.message || clientSecretRes.message || "保存失败"
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "保存失败",
                description: error.message || "保存失败"
            });
        }
        setSavingGithub(false);
    };

    const handleSaveGoogleOAuth = async () => {
        setSavingGoogle(true);
        try {
            const clientIdRes = await api.put('/option/', { key: 'GoogleClientId', value: googleClientId }) as APIResponse;
            const clientSecretRes = await api.put('/option/', { key: 'GoogleClientSecret', value: googleClientSecret }) as APIResponse;

            if (clientIdRes.success && clientSecretRes.success) {
                toast({
                    title: "保存成功",
                    description: "Google OAuth 配置已保存"
                });
            } else {
                toast({
                    variant: "destructive",
                    title: "保存失败",
                    description: clientIdRes.message || clientSecretRes.message || "保存失败"
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "保存失败",
                description: error.message || "保存失败"
            });
        }
        setSavingGoogle(false);
    };

    const handleSaveGitHubOAuthEnabled = async (newValue: boolean) => {
        setSavingGithubEnabled(true);
        try {
            const res = await api.put('/option/', { key: 'GitHubOAuthEnabled', value: newValue.toString() }) as APIResponse;
            if (res.success) {
                toast({
                    title: "保存成功",
                    description: `GitHub OAuth 已${newValue ? '启用' : '禁用'}`
                });
            } else {
                toast({
                    variant: "destructive",
                    title: "保存失败",
                    description: res.message || "保存失败"
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "保存失败",
                description: error.message || "保存失败"
            });
        }
        setSavingGithubEnabled(false);
    };

    const handleSaveGoogleOAuthEnabled = async (newValue: boolean) => {
        setSavingGoogleEnabled(true);
        try {
            const res = await api.put('/option/', { key: 'GoogleOAuthEnabled', value: newValue.toString() }) as APIResponse;
            if (res.success) {
                toast({
                    title: "保存成功",
                    description: `Google OAuth 已${newValue ? '启用' : '禁用'}`
                });
            } else {
                toast({
                    variant: "destructive",
                    title: "保存失败",
                    description: res.message || "保存失败"
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "保存失败",
                description: error.message || "保存失败"
            });
        }
        setSavingGoogleEnabled(false);
    };

    return (
        <div className="w-full max-w-4xl mx-auto space-y-8 p-6">
            <div className="space-y-2">
                <h2 className="text-3xl font-bold tracking-tight">Preferences</h2>
                <p className="text-muted-foreground">管理您的系统配置和登录选项</p>
            </div>

            {/* 通用设置 */}
            <div className="space-y-6">
                <div className="space-y-2">
                    <h3 className="text-xl font-semibold">通用设置</h3>
                    <p className="text-sm text-muted-foreground">配置系统的基本设置</p>
                </div>
                <div className="bg-card border border-border rounded-lg p-6 space-y-4">
                    <div className="space-y-2">
                        <label className="text-sm font-medium">服务器地址</label>
                        <input
                            type="text"
                            className="w-full border border-input rounded-md px-3 py-2 text-sm bg-background"
                            value={serverAddress}
                            onChange={e => setServerAddress(e.target.value)}
                            placeholder="https://one-api.guanzhao12.com"
                            disabled={loading || saving}
                        />
                    </div>
                    <button
                        className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-60 text-sm font-medium"
                        onClick={handleSave}
                        disabled={loading || saving}
                    >
                        {saving ? '更新中...' : '更新服务器地址'}
                    </button>
                    {message && <div className="text-green-600 text-sm mt-2">{message}</div>}
                </div>
            </div>

            {/* 配置登录注册 */}
            <div className="space-y-6">
                <div className="space-y-2">
                    <h3 className="text-xl font-semibold">配置登录注册</h3>
                    <p className="text-sm text-muted-foreground">管理第三方登录选项</p>
                </div>
                <div className="bg-card border border-border rounded-lg p-6 space-y-4">
                    {/* GitHub OAuth 开关 */}
                    <div className="flex items-center justify-between p-4 border border-border rounded-lg bg-muted/30">
                        <div className="flex items-center space-x-3">
                            <input
                                type="checkbox"
                                id="githubOAuthEnabled"
                                checked={githubOAuthEnabled}
                                onChange={(e) => {
                                    const newValue = e.target.checked;
                                    setGithubOAuthEnabled(newValue);
                                    // Use setTimeout to avoid blocking the UI
                                    setTimeout(() => handleSaveGitHubOAuthEnabled(newValue), 0);
                                }}
                                className="h-4 w-4 text-primary focus:ring-primary border-border rounded"
                                disabled={loading || savingGithubEnabled}
                            />
                            <label htmlFor="githubOAuthEnabled" className="text-sm font-medium">
                                允许通过 GitHub 账户登录 & 注册
                            </label>
                        </div>
                        {savingGithubEnabled && (
                            <div className="text-sm text-muted-foreground">保存中...</div>
                        )}
                    </div>

                    {/* Google OAuth 开关 */}
                    <div className="flex items-center justify-between p-4 border border-border rounded-lg bg-muted/30">
                        <div className="flex items-center space-x-3">
                            <input
                                type="checkbox"
                                id="googleOAuthEnabled"
                                checked={googleOAuthEnabled}
                                onChange={(e) => {
                                    const newValue = e.target.checked;
                                    setGoogleOAuthEnabled(newValue);
                                    // Use setTimeout to avoid blocking the UI
                                    setTimeout(() => handleSaveGoogleOAuthEnabled(newValue), 0);
                                }}
                                className="h-4 w-4 text-primary focus:ring-primary border-border rounded"
                                disabled={loading || savingGoogleEnabled}
                            />
                            <label htmlFor="googleOAuthEnabled" className="text-sm font-medium">
                                允许通过 Google 账户登录 & 注册
                            </label>
                        </div>
                        {savingGoogleEnabled && (
                            <div className="text-sm text-muted-foreground">保存中...</div>
                        )}
                    </div>
                </div>
            </div>

            {/* GitHub OAuth 配置 */}
            <div className="space-y-6">
                <div className="space-y-2">
                    <h3 className="text-xl font-semibold">配置 GitHub OAuth App</h3>
                    <p className="text-sm text-muted-foreground">
                        用以支持通过 GitHub 进行登录注册，
                        <a href="https://github.com/settings/applications/new" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline ml-1">
                            点击此处管理你的 GitHub OAuth App
                        </a>
                    </p>
                </div>

                <div className="bg-card border border-border rounded-lg p-6 space-y-4">
                    <div className="bg-info-bg border border-info-border rounded-lg p-4">
                        <div className="flex items-start">
                            <div className="flex-shrink-0">
                                <svg className="h-5 w-5 text-info-foreground" viewBox="0 0 20 20" fill="currentColor">
                                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                                </svg>
                            </div>
                            <div className="ml-3">
                                <p className="text-sm text-info-foreground">
                                    Homepage URL 填 <code className="bg-info-code px-1 rounded font-mono text-xs">{serverAddress || 'https://your-domain.com'}</code>，
                                    Authorization callback URL 填 <code className="bg-info-code px-1 rounded font-mono text-xs">{serverAddress || 'https://your-domain.com'}/oauth/github</code>
                                </p>
                            </div>
                        </div>
                    </div>

                    <div className="space-y-4">
                        <div className="space-y-2">
                            <label className="text-sm font-medium">GitHub Client ID</label>
                            <input
                                type="text"
                                className="w-full border border-input rounded-md px-3 py-2 text-sm bg-background"
                                value={githubClientId}
                                onChange={e => setGithubClientId(e.target.value)}
                                placeholder="输入 GitHub Client ID"
                                disabled={loading || savingGithub}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-sm font-medium">GitHub Client Secret</label>
                            <input
                                type="password"
                                className="w-full border border-input rounded-md px-3 py-2 text-sm bg-background"
                                value={githubClientSecret}
                                onChange={e => setGithubClientSecret(e.target.value)}
                                placeholder="输入 GitHub Client Secret"
                                disabled={loading || savingGithub}
                            />
                        </div>
                        <button
                            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-60 text-sm font-medium"
                            onClick={handleSaveGitHubOAuth}
                            disabled={loading || savingGithub}
                        >
                            {savingGithub ? '保存中...' : '保存 GitHub OAuth 设置'}
                        </button>
                    </div>
                </div>
            </div>

            {/* Google OAuth 配置 */}
            <div className="space-y-6">
                <div className="space-y-2">
                    <h3 className="text-xl font-semibold">配置 Google OAuth App</h3>
                    <p className="text-sm text-muted-foreground">
                        用以支持通过 Google 进行登录注册，
                        <a href="https://console.cloud.google.com/apis/credentials" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline ml-1">
                            点击此处管理你的 Google OAuth App
                        </a>
                    </p>
                </div>

                <div className="bg-card border border-border rounded-lg p-6 space-y-4">
                    <div className="bg-info-bg border border-info-border rounded-lg p-4">
                        <div className="flex items-start">
                            <div className="flex-shrink-0">
                                <svg className="h-5 w-5 text-info-foreground" viewBox="0 0 20 20" fill="currentColor">
                                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                                </svg>
                            </div>
                            <div className="ml-3">
                                <p className="text-sm text-info-foreground">
                                    Authorized JavaScript origins 填 <code className="bg-info-code px-1 rounded font-mono text-xs">{serverAddress || 'https://your-domain.com'}</code>，
                                    Authorized redirect URIs 填 <code className="bg-info-code px-1 rounded font-mono text-xs">{serverAddress || 'https://your-domain.com'}/oauth/google</code>
                                </p>
                            </div>
                        </div>
                    </div>

                    <div className="space-y-4">
                        <div className="space-y-2">
                            <label className="text-sm font-medium">Google Client ID</label>
                            <input
                                type="text"
                                className="w-full border border-input rounded-md px-3 py-2 text-sm bg-background"
                                value={googleClientId}
                                onChange={e => setGoogleClientId(e.target.value)}
                                placeholder="输入 Google Client ID"
                                disabled={loading || savingGoogle}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-sm font-medium">Google Client Secret</label>
                            <input
                                type="password"
                                className="w-full border border-input rounded-md px-3 py-2 text-sm bg-background"
                                value={googleClientSecret}
                                onChange={e => setGoogleClientSecret(e.target.value)}
                                placeholder="输入 Google Client Secret"
                                disabled={loading || savingGoogle}
                            />
                        </div>
                        <button
                            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-60 text-sm font-medium"
                            onClick={handleSaveGoogleOAuth}
                            disabled={loading || savingGoogle}
                        >
                            {savingGoogle ? '保存中...' : '保存 Google OAuth 设置'}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
} 