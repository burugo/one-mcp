import { useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import type { PageOutletContext } from '../App';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableCaption, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import api from '@/utils/api';
import { useToast } from '@/hooks/use-toast';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Terminal } from 'lucide-react';

interface ServiceUtilizationStat {
    service_name: string;
    total_requests: number;
    success_rate: number;
    avg_latency_ms: number;
}

export function AnalyticsPage() {
    // const { setIsOpen } = useOutletContext<PageOutletContext>(); // Ready for future use
    useOutletContext<PageOutletContext>(); // Establish context connection
    const { toast } = useToast(); // Call the hook to get the toast function

    const [utilizationStats, setUtilizationStats] = useState<ServiceUtilizationStat[]>([]);
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);

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

    return (
        <div className="w-full space-y-8">
            <h2 className="text-3xl font-bold tracking-tight mb-8">Analytics</h2>

            <div className="mt-12">
                <h3 className="text-2xl font-bold mb-4">Usage Statistics</h3>
                <Card className="shadow-sm border bg-card/30">
                    <CardHeader>
                        <CardTitle>Service Utilization</CardTitle>
                        <CardDescription>Aggregated performance overview for all services.</CardDescription>
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
                                        <TableHead className="text-right">Total Requests</TableHead>
                                        <TableHead className="text-right">Success Rate</TableHead>
                                        <TableHead className="text-right">Avg. Latency (ms)</TableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {utilizationStats.length === 0 && !isLoading ? (
                                        <TableRow>
                                            <TableCell colSpan={4} className="text-center text-muted-foreground">
                                                No usage statistics available yet.
                                            </TableCell>
                                        </TableRow>
                                    ) : (
                                        utilizationStats.map((stat) => (
                                            <TableRow key={stat.service_name}>
                                                <TableCell className="font-medium">{stat.service_name}</TableCell>
                                                <TableCell className="text-right">{stat.total_requests}</TableCell>
                                                <TableCell className="text-right">{(stat.success_rate * 100).toFixed(2)}%</TableCell>
                                                <TableCell className="text-right">{stat.avg_latency_ms.toFixed(2)}</TableCell>
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