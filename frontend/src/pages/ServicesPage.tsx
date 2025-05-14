
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableCaption, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Search, BarChart, User, PlusCircle } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { useNavigate, useOutletContext } from 'react-router-dom';
import type { PageOutletContext } from '../App';

export function ServicesPage() {
    const { toast } = useToast();
    const { setIsOpen: openDialog } = useOutletContext<PageOutletContext>();
    const navigate = useNavigate();

    return (
        <div className="w-full space-y-8">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h2 className="text-3xl font-bold tracking-tight">MCP Services</h2>
                    <p className="text-muted-foreground mt-1">Manage and configure your multi-cloud platform services</p>
                </div>
                <Button onClick={() => navigate('/market')} className="rounded-full bg-[#7c3aed] hover:bg-[#7c3aed]/90">
                    <PlusCircle className="w-4 h-4 mr-2" /> Add Service
                </Button>
            </div>

            <Tabs defaultValue="all" className="mb-8">
                <TabsList className="w-full max-w-3xl grid grid-cols-3 bg-muted/80 p-1 rounded-lg">
                    <TabsTrigger value="all" className="rounded-md">All Services</TabsTrigger>
                    <TabsTrigger value="active" className="rounded-md">Active</TabsTrigger>
                    <TabsTrigger value="inactive" className="rounded-md">Inactive</TabsTrigger>
                </TabsList>
                <TabsContent value="all">
                    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 mt-6">
                        {[
                            { id: 1, name: "Search Service", status: "Active", description: "AI-powered semantic search", icon: <Search className="w-6 h-6 text-primary" /> },
                            { id: 2, name: "Analytics", status: "Active", description: "Usage tracking and reporting", icon: <BarChart className="w-6 h-6 text-primary" /> },
                            { id: 3, name: "User Management", status: "Inactive", description: "User access and controls", icon: <User className="w-6 h-6 text-primary" /> }
                        ].map(service => (
                            <Card key={service.id} className="border-border shadow-sm hover:shadow transition-shadow duration-200 bg-card/30">
                                <CardHeader>
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center">
                                            <div className="bg-primary/10 p-2 rounded-md mr-3">
                                                {service.icon}
                                            </div>
                                            <div>
                                                <CardTitle className="text-lg">{service.name}</CardTitle>
                                                <CardDescription>
                                                    <span className={`inline-flex items-center px-2 py-1 text-xs rounded-full ${service.status === "Active"
                                                        ? "bg-green-100 text-green-800"
                                                        : "bg-gray-100 text-gray-800"
                                                        }`}>
                                                        {service.status}
                                                    </span>
                                                </CardDescription>
                                            </div>
                                        </div>
                                    </div>
                                </CardHeader>
                                <CardContent>
                                    <p className="text-sm text-muted-foreground">{service.description}</p>
                                </CardContent>
                                <CardFooter className="flex justify-between">
                                    <Button variant="outline" size="sm" onClick={() => openDialog(true)}>Configure</Button>
                                    <Button
                                        variant={service.status === "Active" ? "outline" : "default"}
                                        size="sm"
                                        onClick={() => toast({
                                            title: `Service ${service.status === "Active" ? "Disabled" : "Enabled"}`,
                                            description: `${service.name} is now ${service.status === "Active" ? "inactive" : "active"}.`
                                        })}
                                    >
                                        {service.status === "Active" ? "Disable" : "Enable"}
                                    </Button>
                                </CardFooter>
                            </Card>
                        ))}
                    </div>
                </TabsContent>
                <TabsContent value="active">
                    <p className="text-muted-foreground mt-4">Showing only active services.</p>
                </TabsContent>
                <TabsContent value="inactive">
                    <p className="text-muted-foreground mt-4">Showing only inactive services.</p>
                </TabsContent>
            </Tabs>

            <div className="mt-12">
                <h3 className="text-2xl font-bold mb-4">Usage Statistics</h3>
                <Card className="shadow-sm border bg-card/30">
                    <CardHeader>
                        <CardTitle>Service Utilization</CardTitle>
                        <CardDescription>Performance overview for the last 30 days</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <Table>
                            <TableCaption>A summary of your service usage.</TableCaption>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>Service</TableHead>
                                    <TableHead>Requests</TableHead>
                                    <TableHead>Success Rate</TableHead>
                                    <TableHead className="text-right">Avg. Latency</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                <TableRow>
                                    <TableCell className="font-medium">Search Service</TableCell>
                                    <TableCell>12,423</TableCell>
                                    <TableCell>99.8%</TableCell>
                                    <TableCell className="text-right">132ms</TableCell>
                                </TableRow>
                                <TableRow>
                                    <TableCell className="font-medium">Analytics</TableCell>
                                    <TableCell>5,752</TableCell>
                                    <TableCell>99.4%</TableCell>
                                    <TableCell className="text-right">245ms</TableCell>
                                </TableRow>
                                <TableRow>
                                    <TableCell className="font-medium">User Management</TableCell>
                                    <TableCell>892</TableCell>
                                    <TableCell>100%</TableCell>
                                    <TableCell className="text-right">89ms</TableCell>
                                </TableRow>
                            </TableBody>
                        </Table>
                    </CardContent>
                </Card>
            </div>
        </div>
    );
} 