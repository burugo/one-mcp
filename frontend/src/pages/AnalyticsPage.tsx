import { useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import type { PageOutletContext } from '../App';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableCaption, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import api from '@/utils/api';
import { useToast } from '@/hooks/use-toast';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Terminal, ChevronUp, ChevronDown } from 'lucide-react';

interface ServiceUtilizationStat {
    service_id: number;
    service_name: string;
    display_name: string;
    enabled: boolean;
    today_request_count: number;
    today_avg_latency_ms: number;
}

export function AnalyticsPage() {
    // const { setIsOpen } = useOutletContext<PageOutletContext>(); // Ready for future use
    useOutletContext<PageOutletContext>(); // Establish context connection
    const { toast } = useToast(); // Call the hook to get the toast function

    const [utilizationStats, setUtilizationStats] = useState<ServiceUtilizationStat[]>([]);
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc'); // 默认倒序

    useEffect(() => {
        const fetchUtilizationStats = async () => {
            setIsLoading(true);
            setError(null);
            try {
                const response = await api.get<ServiceUtilizationStat[]>('/analytics/services/utilization');
                if (response.success) {
                    setUtilizationStats(response.data || []);
                } else {
                    setError(response.message || 'Failed to fetch utilization stats');
                    toast({
                        variant: "destructive",
                        title: "Error fetching stats",
                        description: response.message || 'An unknown error occurred',
                    });
                }
            } catch (err: any) {
                const errorMessage = err.message || 'An unexpected error occurred';
                setError(errorMessage);
                toast({
                    variant: "destructive",
                    title: "Network Error",
                    description: errorMessage,
                });
            }
            setIsLoading(false);
        };

        fetchUtilizationStats();
    }, []);

    // 排序和过滤函数
    const sortedAndFilteredStats = utilizationStats
        .filter(stat => stat.enabled) // 只显示启用的服务
        .sort((a, b) => {
            const aRequests = a.today_request_count || 0;
            const bRequests = b.today_request_count || 0;
            return sortOrder === 'desc' ? bRequests - aRequests : aRequests - bRequests;
        });

    // 切换排序方向
    const toggleSortOrder = () => {
        setSortOrder(prev => prev === 'desc' ? 'asc' : 'desc');
    };

    return (
        <div className="w-full space-y-8">
            <h2 className="text-3xl font-bold tracking-tight mb-8">Analytics</h2>

            <div className="mt-12">
                <h3 className="text-2xl font-bold mb-4">Usage Statistics</h3>
                <Card className="shadow-sm border bg-card/30">
                    <CardHeader>
                        <CardTitle>Service Utilization</CardTitle>
                        <CardDescription>
                            Aggregated performance overview for enabled services.
                            Sorted by today's requests ({sortOrder === 'desc' ? 'high to low' : 'low to high'}).
                            Click the column header to change sort order.
                        </CardDescription>
                    </CardHeader>
                    <CardContent>
                        {isLoading && <p className="text-center text-muted-foreground">Loading statistics...</p>}
                        {error && (
                            <Alert variant="destructive" className="mb-4">
                                <Terminal className="h-4 w-4" />
                                <AlertTitle>Error</AlertTitle>
                                <AlertDescription>{error}</AlertDescription>
                            </Alert>
                        )}
                        {!isLoading && !error && (
                            <Table>
                                <TableCaption>A summary of your service usage.</TableCaption>
                                <TableHeader>
                                    <TableRow>
                                        <TableHead>Service</TableHead>
                                        <TableHead className="text-right">Status</TableHead>
                                        <TableHead className="text-right">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={toggleSortOrder}
                                                className="flex items-center gap-1 h-auto p-0 font-medium hover:bg-transparent focus:outline-none focus:ring-0 focus:ring-offset-0 ml-auto"
                                            >
                                                Today's Requests
                                                {sortOrder === 'desc' ? (
                                                    <ChevronDown className="h-4 w-4" />
                                                ) : (
                                                    <ChevronUp className="h-4 w-4" />
                                                )}
                                            </Button>
                                        </TableHead>
                                        <TableHead className="text-right">Today's Avg. Latency (ms)</TableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {sortedAndFilteredStats.length === 0 && !isLoading ? (
                                        <TableRow>
                                            <TableCell colSpan={4} className="text-center text-muted-foreground">
                                                No enabled services with usage statistics available yet.
                                            </TableCell>
                                        </TableRow>
                                    ) : (
                                        sortedAndFilteredStats.map((stat) => (
                                            <TableRow key={stat.service_id}>
                                                <TableCell className="font-medium">{stat.display_name || stat.service_name}</TableCell>
                                                <TableCell className="text-right">
                                                    <span className="px-2 py-1 rounded-full text-xs bg-green-100 text-green-800">
                                                        Enabled
                                                    </span>
                                                </TableCell>
                                                <TableCell className="text-right font-medium">
                                                    {stat.today_request_count || 0}
                                                </TableCell>
                                                <TableCell className="text-right">
                                                    {stat.today_avg_latency_ms ? stat.today_avg_latency_ms.toFixed(2) : '0.00'}
                                                </TableCell>
                                            </TableRow>
                                        ))
                                    )}
                                </TableBody>
                            </Table>
                        )}
                    </CardContent>
                </Card>
            </div>
        </div>
    );
} 