import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Server, Activity, Clock, Database, AlertCircle, CheckCircle, Package, User, Timer } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useEffect, useState } from 'react';
import api from '@/utils/api';
import { useAuth } from '@/contexts/AuthContext';
import { useTranslation } from 'react-i18next';

interface SystemOverview {
    total_services: number;
    enabled_services: number;
    healthy_services: number;
    unhealthy_services: number;
    today_total_requests: number;
    today_avg_response_time_ms: number;
}

interface SystemStatus {
    start_time: number; // Unix timestamp
}

export function DashboardPage() {
    const navigate = useNavigate();
    const { currentUser } = useAuth();
    const { t, i18n } = useTranslation();

    // State for API data
    const [systemOverview, setSystemOverview] = useState<SystemOverview | null>(null);
    const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Helper function to format uptime
    const formatUptime = (startTime: number): string => {
        const now = new Date();
        const start = new Date(startTime * 1000); // Convert Unix timestamp to milliseconds
        const diffMs = now.getTime() - start.getTime();

        const days = Math.floor(diffMs / (1000 * 60 * 60 * 24));
        const hours = Math.floor((diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
        const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));

        if (i18n.language === 'zh-CN') {
            if (days > 0) {
                return `${days}天 ${hours}小时 ${minutes}分钟`;
            } else if (hours > 0) {
                return `${hours}小时 ${minutes}分钟`;
            } else {
                return `${minutes}分钟`;
            }
        } else {
            if (days > 0) {
                return `${days}d ${hours}h ${minutes}m`;
            } else if (hours > 0) {
                return `${hours}h ${minutes}m`;
            } else {
                return `${minutes}m`;
            }
        }
    };



    // Fetch data from APIs
    useEffect(() => {
        const fetchData = async () => {
            try {
                setLoading(true);
                setError(null);

                // Fetch system overview and status in parallel
                const [overviewResponse, statusResponse] = await Promise.all([
                    api.get('/analytics/system/overview'),
                    api.get('/status')
                ]);

                setSystemOverview(overviewResponse.data);
                setSystemStatus(statusResponse.data);
            } catch (err: any) {
                console.error('Error fetching dashboard data:', err);
                setError(err.response?.data?.message || t('dashboard.fetchDataFailed'));
            } finally {
                setLoading(false);
            }
        };

        fetchData();
    }, []);

    // Get health statistics from system overview data
    const healthStats = {
        enabledCount: systemOverview?.enabled_services || 0,
        healthyCount: systemOverview?.healthy_services || 0,
        unhealthyCount: systemOverview?.unhealthy_services || 0
    };

    if (loading) {
        return (
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-8 py-8">
                <h2 className="text-3xl font-bold tracking-tight mb-2">{t('dashboard.title')}</h2>
                <div className="animate-pulse space-y-8">
                    <div className="h-32 bg-gray-200 rounded-lg"></div>
                    <div className="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
                        {[...Array(4)].map((_, i) => (
                            <div key={i} className="h-24 bg-gray-200 rounded-lg"></div>
                        ))}
                    </div>
                    <div className="grid gap-6 grid-cols-1 lg:grid-cols-2">
                        <div className="h-64 bg-gray-200 rounded-lg"></div>
                        <div className="h-64 bg-gray-200 rounded-lg"></div>
                    </div>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-8 py-8">
                <h2 className="text-3xl font-bold tracking-tight mb-2">{t('dashboard.title')}</h2>
                <Card className="border-red-200 bg-red-50">
                    <CardContent className="pt-6">
                        <div className="flex items-center space-x-2">
                            <AlertCircle className="h-5 w-5 text-red-500" />
                            <span className="text-red-700">{t('dashboard.error')}: {error}</span>
                        </div>
                    </CardContent>
                </Card>
            </div>
        );
    }

    return (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-8 py-8">
            <h2 className="text-3xl font-bold tracking-tight mb-2">{t('dashboard.title')}</h2>

            {/* 欢迎卡片 */}
            <Card className="border-border bg-gradient-to-br from-primary/5 to-primary/10">
                <CardHeader className="text-center">
                    <CardTitle className="text-2xl">{t('dashboard.welcome')}</CardTitle>
                    <CardDescription className="text-base">{t('dashboard.description')}</CardDescription>
                </CardHeader>
                <CardContent className="text-center max-w-2xl mx-auto">
                    <p className="text-muted-foreground">
                        {t('dashboard.content')}
                    </p>
                </CardContent>
            </Card>

            {/* 统计卡片 */}
            <div className="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
                <Card className="border bg-card/30">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">{t('dashboard.activeServices')}</CardTitle>
                        <Server className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="pt-2">
                        <div className="text-3xl font-bold mb-1">{systemOverview?.enabled_services || 0}</div>
                        <p className="text-xs text-muted-foreground">{t('dashboard.activeServicesDesc')}</p>
                    </CardContent>
                </Card>

                <Card className="border bg-card/30">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">{t('dashboard.todayRequests')}</CardTitle>
                        <Activity className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="pt-2">
                        <div className="text-3xl font-bold mb-1">{systemOverview?.today_total_requests || 0}</div>
                        <p className="text-xs text-muted-foreground">{t('dashboard.todayRequestsDesc')}</p>
                    </CardContent>
                </Card>

                <Card className="border bg-card/30">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">{t('dashboard.avgResponseTime')}</CardTitle>
                        <Clock className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="pt-2">
                        <div className="text-3xl font-bold mb-1">
                            {systemOverview?.today_avg_response_time_ms
                                ? `${Math.round(systemOverview.today_avg_response_time_ms)}ms`
                                : '0ms'
                            }
                        </div>
                        <p className="text-xs text-muted-foreground">{t('dashboard.avgResponseTimeDesc')}</p>
                    </CardContent>
                </Card>

                <Card className="border bg-card/30">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">{t('dashboard.remainingCredits')}</CardTitle>
                        <Database className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="pt-2">
                        <div className="text-3xl font-bold mb-1">0</div>
                        <p className="text-xs text-muted-foreground">{t('dashboard.remainingCreditsDesc')}</p>
                    </CardContent>
                </Card>
            </div>

            {/* 系统状态和快速访问 */}
            <div className="grid gap-6 grid-cols-1 lg:grid-cols-2">
                <Card className="col-span-1 bg-card/30 border">
                    <CardHeader>
                        <CardTitle>{t('dashboard.systemStatus')}</CardTitle>
                        <CardDescription>{t('dashboard.systemStatusDesc')}</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-4">
                            {/* 系统运行时间 */}
                            <div className="flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                    <Timer className="h-5 w-5 text-blue-500" />
                                    <span>{t('dashboard.systemUptime')}</span>
                                </div>
                                <div className="flex items-center">
                                    <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded-full">
                                        {systemStatus?.start_time ? formatUptime(systemStatus.start_time) : t('dashboard.unknown')}
                                    </span>
                                </div>
                            </div>

                            {/* 服务健康概览 */}
                            <div className="flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                    <Server className="h-5 w-5 text-green-500" />
                                    <span>{t('dashboard.totalServices')}</span>
                                </div>
                                <div className="flex items-center">
                                    <span className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded-full">
                                        {healthStats.enabledCount} {t('dashboard.enabled')}
                                    </span>
                                </div>
                            </div>

                            <div className="flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                    <CheckCircle className="h-5 w-5 text-green-500" />
                                    <span>{t('dashboard.healthyServices')}</span>
                                </div>
                                <div className="flex items-center">
                                    <span className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded-full">
                                        {healthStats.healthyCount}
                                    </span>
                                </div>
                            </div>

                            {healthStats.unhealthyCount > 0 && (
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center space-x-2">
                                        <AlertCircle className="h-5 w-5 text-red-500" />
                                        <span>{t('dashboard.unhealthyServices')}</span>
                                    </div>
                                    <div className="flex items-center">
                                        <span className="text-xs bg-red-100 text-red-800 px-2 py-1 rounded-full">
                                            {healthStats.unhealthyCount}
                                        </span>
                                    </div>
                                </div>
                            )}
                        </div>
                    </CardContent>
                </Card>

                <Card className="col-span-1 bg-card/30 border">
                    <CardHeader>
                        <CardTitle>{t('dashboard.quickActions')}</CardTitle>
                        <CardDescription>{t('dashboard.quickActionsDesc')}</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <div className="grid gap-4 grid-cols-1 sm:grid-cols-2">
                            <Button
                                variant="outline"
                                className="flex justify-start space-x-2 h-auto py-3"
                                onClick={() => navigate('/services')}
                            >
                                <Server className="h-5 w-5 text-primary" />
                                <div className="text-left">
                                    <p className="font-medium">{t('dashboard.manageServices')}</p>
                                    <p className="text-xs text-muted-foreground">{t('dashboard.manageServicesDesc')}</p>
                                </div>
                            </Button>

                            <Button
                                variant="outline"
                                className="flex justify-start space-x-2 h-auto py-3"
                                onClick={() => navigate('/analytics')}
                            >
                                <Activity className="h-5 w-5 text-primary" />
                                <div className="text-left">
                                    <p className="font-medium">{t('dashboard.viewAnalytics')}</p>
                                    <p className="text-xs text-muted-foreground">{t('dashboard.viewAnalyticsDesc')}</p>
                                </div>
                            </Button>

                            <Button
                                variant="outline"
                                className="flex justify-start space-x-2 h-auto py-3"
                                onClick={() => navigate('/profile')}
                            >
                                <User className="h-5 w-5 text-primary" />
                                <div className="text-left">
                                    <p className="font-medium">{t('dashboard.userSettings')}</p>
                                    <p className="text-xs text-muted-foreground">{t('dashboard.userSettingsDesc')}</p>
                                </div>
                            </Button>
                            {currentUser?.role && currentUser.role >= 10 && (
                                <Button
                                    variant="outline"
                                    className="flex justify-start space-x-2 h-auto py-3"
                                    onClick={() => navigate('/market')}
                                >
                                    <Package className="h-5 w-5 text-primary" />
                                    <div className="text-left">
                                        <p className="font-medium">{t('dashboard.installService')}</p>
                                        <p className="text-xs text-muted-foreground">{t('dashboard.installServiceDesc')}</p>
                                    </div>
                                </Button>
                            )}
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* 最近活动日志 */}
            <Card className="border bg-card/30">
                <CardHeader>
                    <CardTitle>{t('dashboard.recentActivity')}</CardTitle>
                    <CardDescription>{t('dashboard.recentActivityDesc')}</CardDescription>
                </CardHeader>
                <CardContent>
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead className="w-[120px]">{t('dashboard.time')}</TableHead>
                                <TableHead>{t('dashboard.event')}</TableHead>
                                <TableHead>{t('dashboard.service')}</TableHead>
                                <TableHead className="text-right">{t('dashboard.status')}</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            <TableRow>
                                <TableCell className="font-medium">{t('dashboard.today')} 14:32</TableCell>
                                <TableCell>{t('dashboard.serviceStarted')}</TableCell>
                                <TableCell>{t('dashboard.searchService')}</TableCell>
                                <TableCell className="text-right text-green-600">{t('dashboard.success')}</TableCell>
                            </TableRow>
                            <TableRow>
                                <TableCell className="font-medium">{t('dashboard.today')} 13:15</TableCell>
                                <TableCell>{t('dashboard.configUpdated')}</TableCell>
                                <TableCell>{t('dashboard.analytics')}</TableCell>
                                <TableCell className="text-right text-green-600">{t('dashboard.success')}</TableCell>
                            </TableRow>
                            <TableRow>
                                <TableCell className="font-medium">{t('dashboard.today')} 10:41</TableCell>
                                <TableCell>{t('dashboard.apiKeyGenerated')}</TableCell>
                                <TableCell>{t('dashboard.system')}</TableCell>
                                <TableCell className="text-right text-green-600">{t('dashboard.success')}</TableCell>
                            </TableRow>
                            <TableRow>
                                <TableCell className="font-medium">{t('dashboard.yesterday')}</TableCell>
                                <TableCell>{t('dashboard.serviceRestarted')}</TableCell>
                                <TableCell>{t('dashboard.userManagement')}</TableCell>
                                <TableCell className="text-right text-amber-600">{t('dashboard.warning')}</TableCell>
                            </TableRow>
                        </TableBody>
                    </Table>
                </CardContent>
            </Card>
        </div>
    );
} 