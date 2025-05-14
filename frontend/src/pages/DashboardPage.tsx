import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Server, Activity, Clock, Database, AlertCircle, CheckCircle, Package, User } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

export function DashboardPage() {
    const navigate = useNavigate();

    return (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 space-y-8 py-8">
            <h2 className="text-3xl font-bold tracking-tight mb-2">Dashboard</h2>

            {/* 欢迎卡片 */}
            <Card className="border-border bg-gradient-to-br from-primary/5 to-primary/10">
                <CardHeader className="text-center">
                    <CardTitle className="text-2xl">Welcome to One MCP</CardTitle>
                    <CardDescription className="text-base">Your multi-cloud platform management center</CardDescription>
                </CardHeader>
                <CardContent className="text-center max-w-2xl mx-auto">
                    <p className="text-muted-foreground">
                        Manage your MCP services, monitor performance, and configure your multi-cloud platform from a single interface.
                    </p>
                </CardContent>
            </Card>

            {/* 统计卡片 */}
            <div className="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
                <Card className="border bg-card/30">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Active Services</CardTitle>
                        <Server className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="pt-2">
                        <div className="text-3xl font-bold mb-1">2</div>
                        <p className="text-xs text-muted-foreground">+1 from last week</p>
                    </CardContent>
                </Card>

                <Card className="border bg-card/30">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">API Requests</CardTitle>
                        <Activity className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="pt-2">
                        <div className="text-3xl font-bold mb-1">18,472</div>
                        <p className="text-xs text-muted-foreground">+12% from last month</p>
                    </CardContent>
                </Card>

                <Card className="border bg-card/30">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Avg. Response Time</CardTitle>
                        <Clock className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="pt-2">
                        <div className="text-3xl font-bold mb-1">154ms</div>
                        <p className="text-xs text-muted-foreground">-8ms from last week</p>
                    </CardContent>
                </Card>

                <Card className="border bg-card/30">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Remaining Credits</CardTitle>
                        <Database className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent className="pt-2">
                        <div className="text-3xl font-bold mb-1">5,842</div>
                        <p className="text-xs text-muted-foreground">Updated just now</p>
                    </CardContent>
                </Card>
            </div>

            {/* 系统状态和快速访问 */}
            <div className="grid gap-6 grid-cols-1 lg:grid-cols-2">
                <Card className="col-span-1 bg-card/30 border">
                    <CardHeader>
                        <CardTitle>System Status</CardTitle>
                        <CardDescription>Real-time status of your services</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                    <CheckCircle className="h-5 w-5 text-green-500" />
                                    <span>Search Service</span>
                                </div>
                                <div className="flex items-center">
                                    <span className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded-full">Operational</span>
                                </div>
                            </div>

                            <div className="flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                    <CheckCircle className="h-5 w-5 text-green-500" />
                                    <span>Analytics Service</span>
                                </div>
                                <div className="flex items-center">
                                    <span className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded-full">Operational</span>
                                </div>
                            </div>

                            <div className="flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                    <AlertCircle className="h-5 w-5 text-amber-500" />
                                    <span>User Management</span>
                                </div>
                                <div className="flex items-center">
                                    <span className="text-xs bg-amber-100 text-amber-800 px-2 py-1 rounded-full">Degraded</span>
                                </div>
                            </div>

                            <div className="flex items-center justify-between">
                                <div className="flex items-center space-x-2">
                                    <Clock className="h-5 w-5 text-blue-500" />
                                    <span>Pending Deployments</span>
                                </div>
                                <div className="flex items-center">
                                    <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded-full">1 Pending</span>
                                </div>
                            </div>
                        </div>
                    </CardContent>
                </Card>

                <Card className="col-span-1 bg-card/30 border">
                    <CardHeader>
                        <CardTitle>Quick Actions</CardTitle>
                        <CardDescription>Get started with your MCP platform</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <div className="grid gap-4 grid-cols-1 sm:grid-cols-2">
                            <Button
                                variant="outline"
                                className="flex justify-start space-x-2 h-auto py-3"
                                onClick={() => navigate('/market')}
                            >
                                <Package className="h-5 w-5 text-primary" />
                                <div className="text-left">
                                    <p className="font-medium">Install Service</p>
                                    <p className="text-xs text-muted-foreground">Add new MCP services</p>
                                </div>
                            </Button>

                            <Button
                                variant="outline"
                                className="flex justify-start space-x-2 h-auto py-3"
                                onClick={() => navigate('/services')}
                            >
                                <Server className="h-5 w-5 text-primary" />
                                <div className="text-left">
                                    <p className="font-medium">Manage Services</p>
                                    <p className="text-xs text-muted-foreground">Configure existing services</p>
                                </div>
                            </Button>

                            <Button
                                variant="outline"
                                className="flex justify-start space-x-2 h-auto py-3"
                                onClick={() => navigate('/analytics')}
                            >
                                <Activity className="h-5 w-5 text-primary" />
                                <div className="text-left">
                                    <p className="font-medium">View Analytics</p>
                                    <p className="text-xs text-muted-foreground">Check performance metrics</p>
                                </div>
                            </Button>

                            <Button
                                variant="outline"
                                className="flex justify-start space-x-2 h-auto py-3"
                                onClick={() => navigate('/profile')}
                            >
                                <User className="h-5 w-5 text-primary" />
                                <div className="text-left">
                                    <p className="font-medium">User Settings</p>
                                    <p className="text-xs text-muted-foreground">Manage your account</p>
                                </div>
                            </Button>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* 最近活动日志 */}
            <Card className="border bg-card/30">
                <CardHeader>
                    <CardTitle>Recent Activity</CardTitle>
                    <CardDescription>Latest actions and system events</CardDescription>
                </CardHeader>
                <CardContent>
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead className="w-[120px]">Time</TableHead>
                                <TableHead>Event</TableHead>
                                <TableHead>Service</TableHead>
                                <TableHead className="text-right">Status</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            <TableRow>
                                <TableCell className="font-medium">Today 14:32</TableCell>
                                <TableCell>Service started</TableCell>
                                <TableCell>Search Service</TableCell>
                                <TableCell className="text-right text-green-600">Success</TableCell>
                            </TableRow>
                            <TableRow>
                                <TableCell className="font-medium">Today 13:15</TableCell>
                                <TableCell>Config updated</TableCell>
                                <TableCell>Analytics</TableCell>
                                <TableCell className="text-right text-green-600">Success</TableCell>
                            </TableRow>
                            <TableRow>
                                <TableCell className="font-medium">Today 10:41</TableCell>
                                <TableCell>API key generated</TableCell>
                                <TableCell>System</TableCell>
                                <TableCell className="text-right text-green-600">Success</TableCell>
                            </TableRow>
                            <TableRow>
                                <TableCell className="font-medium">Yesterday</TableCell>
                                <TableCell>Service restarted</TableCell>
                                <TableCell>User Management</TableCell>
                                <TableCell className="text-right text-amber-600">Warning</TableCell>
                            </TableRow>
                        </TableBody>
                    </Table>
                </CardContent>
            </Card>
        </div>
    );
} 