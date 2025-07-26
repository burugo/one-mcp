import { useEffect, useState } from 'react';
import api, { APIResponse } from '@/utils/api';
import { useServerAddressStore } from '@/hooks/useServerAddress';
import { useToast } from '@/hooks/use-toast';
import { useTranslation } from 'react-i18next';

export function PreferencesPage() {
    const { t } = useTranslation();
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

    // Stdio startup strategy
    const [stdioStartupStrategy, setStdioStartupStrategy] = useState('boot');
    const [savingStartupStrategy, setSavingStartupStrategy] = useState(false);

    useEffect(() => {
        setLoading(true);
        Promise.all([
            fetchServerAddress(),
            loadSettings()
        ]).finally(() => setLoading(false));
    }, [fetchServerAddress]);

    const loadSettings = async () => {
        try {
            const res = await api.get('/option/') as APIResponse;
            if (res.success && Array.isArray(res.data)) {
                const githubClientIdOption = res.data.find((item: any) => item.key === 'GitHubClientId');
                const githubClientSecretOption = res.data.find((item: any) => item.key === 'GitHubClientSecret');
                const googleClientIdOption = res.data.find((item: any) => item.key === 'GoogleClientId');
                const googleClientSecretOption = res.data.find((item: any) => item.key === 'GoogleClientSecret');
                const githubOAuthEnabledOption = res.data.find((item: any) => item.key === 'GitHubOAuthEnabled');
                const googleOAuthEnabledOption = res.data.find((item: any) => item.key === 'GoogleOAuthEnabled');
                const startupStrategyOption = res.data.find((item: any) => item.key === 'StdioServiceStartupStrategy');

                if (githubClientIdOption) setGithubClientId(githubClientIdOption.value);
                if (githubClientSecretOption) setGithubClientSecret(githubClientSecretOption.value);
                if (googleClientIdOption) setGoogleClientId(googleClientIdOption.value);
                if (googleClientSecretOption) setGoogleClientSecret(googleClientSecretOption.value);
                if (githubOAuthEnabledOption) setGithubOAuthEnabled(githubOAuthEnabledOption.value === 'true');
                if (googleOAuthEnabledOption) setGoogleOAuthEnabled(googleOAuthEnabledOption.value === 'true');
                if (startupStrategyOption) setStdioStartupStrategy(startupStrategyOption.value);
            }
        } catch (error) {
            console.error('Failed to load settings:', error);
        }
    };

    const handleSave = async () => {
        setSaving(true);
        setMessage('');
        const clean = serverAddress.replace(/\/$/, '');
        const res = await api.put('/option/', { key: 'ServerAddress', value: clean }) as APIResponse;
        if (res.success) {
            setMessage(t('preferences.messages.saveSuccess'));
            setServerAddress(clean); // 立即同步到全局
            sessionStorage.setItem('server_address', clean); // 同步更新缓存
        } else {
            setMessage(res.message || t('preferences.messages.saveFailed'));
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
                    title: t('preferences.messages.saveSuccess'),
                    description: t('preferences.messages.githubOAuthSaved')
                });
            } else {
                toast({
                    variant: "destructive",
                    title: t('preferences.messages.saveFailed'),
                    description: clientIdRes.message || clientSecretRes.message || t('preferences.messages.saveFailed')
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: t('preferences.messages.saveFailed'),
                description: error.message || t('preferences.messages.saveFailed')
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
                    title: t('preferences.messages.saveSuccess'),
                    description: t('preferences.messages.googleOAuthSaved')
                });
            } else {
                toast({
                    variant: "destructive",
                    title: t('preferences.messages.saveFailed'),
                    description: clientIdRes.message || clientSecretRes.message || t('preferences.messages.saveFailed')
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: t('preferences.messages.saveFailed'),
                description: error.message || t('preferences.messages.saveFailed')
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
                    title: t('preferences.messages.saveSuccess'),
                    description: newValue ? t('preferences.messages.githubOAuthEnabled') : t('preferences.messages.githubOAuthDisabled')
                });
            } else {
                toast({
                    variant: "destructive",
                    title: t('preferences.messages.saveFailed'),
                    description: res.message || t('preferences.messages.saveFailed')
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: t('preferences.messages.saveFailed'),
                description: error.message || t('preferences.messages.saveFailed')
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
                    title: t('preferences.messages.saveSuccess'),
                    description: newValue ? t('preferences.messages.googleOAuthEnabled') : t('preferences.messages.googleOAuthDisabled')
                });
            } else {
                toast({
                    variant: "destructive",
                    title: t('preferences.messages.saveFailed'),
                    description: res.message || t('preferences.messages.saveFailed')
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: t('preferences.messages.saveFailed'),
                description: error.message || t('preferences.messages.saveFailed')
            });
        }
        setSavingGoogleEnabled(false);
    };

    const handleSaveStartupStrategy = async (newValue: string) => {
        setSavingStartupStrategy(true);
        try {
            const res = await api.put('/option/', { key: 'StdioServiceStartupStrategy', value: newValue }) as APIResponse;
            if (res.success) {
                toast({
                    title: t('preferences.messages.saveSuccess'),
                    description: t('preferences.messages.startupStrategyUpdated')
                });
            } else {
                toast({
                    variant: "destructive",
                    title: t('preferences.messages.saveFailed'),
                    description: res.message || t('preferences.messages.saveFailed')
                });
            }
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: t('preferences.messages.saveFailed'),
                description: error.message || t('preferences.messages.saveFailed')
            });
        }
        setSavingStartupStrategy(false);
    };

    return (
        <div className="w-full max-w-4xl mx-auto space-y-8 p-6">
            <div className="space-y-2">
                <h2 className="text-3xl font-bold tracking-tight">{t('preferences.title')}</h2>
                <p className="text-muted-foreground">{t('preferences.description')}</p>
            </div>

            {/* 通用设置 */}
            <div className="space-y-6">
                <div className="space-y-2">
                    <h3 className="text-xl font-semibold">{t('preferences.general')}</h3>
                    <p className="text-sm text-muted-foreground">{t('preferences.generalDesc')}</p>
                </div>
                <div className="bg-card border border-border rounded-lg p-6 space-y-4">
                    <div className="space-y-2">
                        <label className="text-sm font-medium">{t('preferences.form.serverAddress')}</label>
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
                        {saving ? t('preferences.actions.saving') : t('preferences.actions.save')}
                    </button>
                    {message && <div className="text-green-600 text-sm mt-2">{message}</div>}
                </div>
            </div>

            {/* Stdio 服务启动策略 */}
            <div className="space-y-6">
                <div className="space-y-2">
                    <h3 className="text-xl font-semibold">{t('preferences.startupStrategy')}</h3>
                    <p className="text-sm text-muted-foreground">{t('preferences.startupStrategyDesc')}</p>
                </div>
                <div className="bg-card border border-border rounded-lg p-6 space-y-4">
                    <div className="space-y-4">
                        <div className="flex items-center space-x-3">
                            <input
                                type="radio"
                                id="strategyBoot"
                                name="startupStrategy"
                                value="boot"
                                checked={stdioStartupStrategy === 'boot'}
                                onChange={(e) => {
                                    const newValue = e.target.value;
                                    setStdioStartupStrategy(newValue);
                                    setTimeout(() => handleSaveStartupStrategy(newValue), 0);
                                }}
                                className="h-4 w-4 text-primary focus:ring-primary border-border"
                                disabled={loading || savingStartupStrategy}
                            />
                            <div className="flex-1">
                                <label htmlFor="strategyBoot" className="text-sm font-medium cursor-pointer">
                                    {t('preferences.form.startupStrategyBoot')}
                                </label>
                                <p className="text-xs text-muted-foreground mt-1">
                                    {t('preferences.form.startupStrategyBootDesc')}
                                </p>
                            </div>
                        </div>
                        <div className="flex items-center space-x-3">
                            <input
                                type="radio"
                                id="strategyDemand"
                                name="startupStrategy"
                                value="demand"
                                checked={stdioStartupStrategy === 'demand'}
                                onChange={(e) => {
                                    const newValue = e.target.value;
                                    setStdioStartupStrategy(newValue);
                                    setTimeout(() => handleSaveStartupStrategy(newValue), 0);
                                }}
                                className="h-4 w-4 text-primary focus:ring-primary border-border"
                                disabled={loading || savingStartupStrategy}
                            />
                            <div className="flex-1">
                                <label htmlFor="strategyDemand" className="text-sm font-medium cursor-pointer">
                                    {t('preferences.form.startupStrategyDemand')}
                                </label>
                                <p className="text-xs text-muted-foreground mt-1">
                                    {t('preferences.form.startupStrategyDemandDesc')}
                                </p>
                            </div>
                        </div>
                        {savingStartupStrategy && (
                            <div className="text-sm text-muted-foreground">{t('preferences.actions.saving')}</div>
                        )}
                    </div>
                </div>
            </div>

            {/* 配置登录注册 */}
            <div className="space-y-6">
                <div className="space-y-2">
                    <h3 className="text-xl font-semibold">{t('preferences.oauth')}</h3>
                    <p className="text-sm text-muted-foreground">{t('preferences.oauthDesc')}</p>
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
                                {t('preferences.form.enableGithubOAuth')}
                            </label>
                        </div>
                        {savingGithubEnabled && (
                            <div className="text-sm text-muted-foreground">{t('preferences.actions.saving')}</div>
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
                                {t('preferences.form.enableGoogleOAuth')}
                            </label>
                        </div>
                        {savingGoogleEnabled && (
                            <div className="text-sm text-muted-foreground">{t('preferences.actions.saving')}</div>
                        )}
                    </div>
                </div>
            </div>

            {/* GitHub OAuth 配置 */}
            <div className="space-y-6">
                <div className="space-y-2">
                    <h3 className="text-xl font-semibold">{t('preferences.github')}</h3>
                    <p className="text-sm text-muted-foreground">
                        {t('preferences.githubDesc')}，
                        <a href="https://github.com/settings/applications/new" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline ml-1">
                            {t('preferences.githubLinkText')}
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
                                    {t('preferences.instructions.githubHomepage')} <code className="bg-info-code px-1 rounded font-mono text-xs">{serverAddress || 'https://your-domain.com'}</code>{t('preferences.instructions.githubCallback')} <code className="bg-info-code px-1 rounded font-mono text-xs">{serverAddress || 'https://your-domain.com'}/oauth/github</code>
                                </p>
                            </div>
                        </div>
                    </div>

                    <div className="space-y-4">
                        <div className="space-y-2">
                            <label className="text-sm font-medium">{t('preferences.form.githubClientId')}</label>
                            <input
                                type="text"
                                className="w-full border border-input rounded-md px-3 py-2 text-sm bg-background"
                                value={githubClientId}
                                onChange={e => setGithubClientId(e.target.value)}
                                placeholder={t('preferences.form.githubClientIdPlaceholder')}
                                disabled={loading || savingGithub}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-sm font-medium">{t('preferences.form.githubClientSecret')}</label>
                            <input
                                type="password"
                                className="w-full border border-input rounded-md px-3 py-2 text-sm bg-background"
                                value={githubClientSecret}
                                onChange={e => setGithubClientSecret(e.target.value)}
                                placeholder={t('preferences.form.githubClientSecretPlaceholder')}
                                disabled={loading || savingGithub}
                            />
                        </div>
                        <button
                            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-60 text-sm font-medium"
                            onClick={handleSaveGitHubOAuth}
                            disabled={loading || savingGithub}
                        >
                            {savingGithub ? t('preferences.actions.saving') : t('preferences.actions.save')}
                        </button>
                    </div>
                </div>
            </div>

            {/* Google OAuth 配置 */}
            <div className="space-y-6">
                <div className="space-y-2">
                    <h3 className="text-xl font-semibold">{t('preferences.google')}</h3>
                    <p className="text-sm text-muted-foreground">
                        {t('preferences.googleDesc')}，
                        <a href="https://console.cloud.google.com/apis/credentials" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline ml-1">
                            {t('preferences.googleLinkText')}
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
                                    {t('preferences.instructions.googleOrigins')} <code className="bg-info-code px-1 rounded font-mono text-xs">{serverAddress || 'https://your-domain.com'}</code>{t('preferences.instructions.googleRedirect')} <code className="bg-info-code px-1 rounded font-mono text-xs">{serverAddress || 'https://your-domain.com'}/oauth/google</code>
                                </p>
                            </div>
                        </div>
                    </div>

                    <div className="space-y-4">
                        <div className="space-y-2">
                            <label className="text-sm font-medium">{t('preferences.form.googleClientId')}</label>
                            <input
                                type="text"
                                className="w-full border border-input rounded-md px-3 py-2 text-sm bg-background"
                                value={googleClientId}
                                onChange={e => setGoogleClientId(e.target.value)}
                                placeholder={t('preferences.form.googleClientIdPlaceholder')}
                                disabled={loading || savingGoogle}
                            />
                        </div>
                        <div className="space-y-2">
                            <label className="text-sm font-medium">{t('preferences.form.googleClientSecret')}</label>
                            <input
                                type="password"
                                className="w-full border border-input rounded-md px-3 py-2 text-sm bg-background"
                                value={googleClientSecret}
                                onChange={e => setGoogleClientSecret(e.target.value)}
                                placeholder={t('preferences.form.googleClientSecretPlaceholder')}
                                disabled={loading || savingGoogle}
                            />
                        </div>
                        <button
                            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-60 text-sm font-medium"
                            onClick={handleSaveGoogleOAuth}
                            disabled={loading || savingGoogle}
                        >
                            {savingGoogle ? t('preferences.actions.saving') : t('preferences.actions.save')}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
} 