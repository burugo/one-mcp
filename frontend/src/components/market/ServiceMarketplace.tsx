import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { useToast } from '@/hooks/use-toast';
import { Search, Package, Star, Download, Filter } from 'lucide-react';

// 服务类型定义
type ServiceType = {
    id: string;
    name: string;
    description: string;
    version: string;
    source: 'npm' | 'pypi' | 'recommended';
    author: string;
    downloads: number;
    stars: number;
    icon?: React.ReactNode;
};

// 示例数据
const mockServices: ServiceType[] = [
    {
        id: '1',
        name: '@modelcontextprotocol/server-airtable',
        description: 'MCP server for Airtable integration',
        version: '1.0.4',
        source: 'npm',
        author: 'MCP Team',
        downloads: 12540,
        stars: 145,
        icon: <Package className="w-6 h-6 text-primary" />
    },
    {
        id: '2',
        name: '@modelcontextprotocol/server-notion',
        description: 'Connect your Notion workspace to AI models',
        version: '0.9.2',
        source: 'npm',
        author: 'MCP Team',
        downloads: 8720,
        stars: 132,
        icon: <Package className="w-6 h-6 text-primary" />
    },
    {
        id: '3',
        name: 'mcp-github-connector',
        description: 'Browse and search GitHub repositories with AI',
        version: '1.2.0',
        source: 'npm',
        author: 'Community',
        downloads: 6940,
        stars: 89,
        icon: <Package className="w-6 h-6 text-primary" />
    },
    {
        id: '4',
        name: 'mcp-google-drive',
        description: 'Access Google Drive documents with MCP',
        version: '0.8.5',
        source: 'recommended',
        author: 'Official',
        downloads: 15840,
        stars: 201,
        icon: <Package className="w-6 h-6 text-primary" />
    },
];

export function ServiceMarketplace({ onSelectService }: { onSelectService: (serviceId: string) => void }) {
    const { toast } = useToast();
    const [services, setServices] = useState<ServiceType[]>(mockServices);
    const [searchTerm, setSearchTerm] = useState('');
    const [isLoading, setIsLoading] = useState(false);

    // 搜索处理函数
    const handleSearch = () => {
        setIsLoading(true);

        // 这里可以实现真实的API调用
        setTimeout(() => {
            const filteredServices = mockServices.filter(
                service => service.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                    service.description.toLowerCase().includes(searchTerm.toLowerCase())
            );
            setServices(filteredServices);
            setIsLoading(false);
        }, 500);
    };

    // 处理搜索框按下回车
    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            handleSearch();
        }
    };

    // 查看服务详情
    const viewServiceDetails = (serviceId: string) => {
        // 调用传入的回调函数
        onSelectService(serviceId);
    };

    return (
        <div className="flex-1 space-y-6">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h2 className="text-3xl font-bold tracking-tight">Service Marketplace</h2>
                    <p className="text-muted-foreground mt-1">
                        Discover and install MCP services from various sources
                    </p>
                </div>
            </div>

            {/* 搜索和过滤部分 */}
            <div className="flex flex-col md:flex-row gap-4 mb-6">
                <div className="relative flex-grow">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                        className="pl-10 bg-muted/40 w-full"
                        placeholder="Search for MCP services..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        onKeyDown={handleKeyDown}
                    />
                </div>
                <Button onClick={handleSearch} disabled={isLoading}>
                    {isLoading ? 'Searching...' : 'Search'}
                </Button>
                <Button variant="outline" className="flex items-center gap-2">
                    <Filter className="h-4 w-4" /> Filter
                </Button>
            </div>

            {/* 选项卡分类 */}
            <Tabs defaultValue="all" className="mb-6">
                <TabsList className="w-full max-w-lg grid grid-cols-4 gap-4">
                    <TabsTrigger value="all" className="px-4">All</TabsTrigger>
                    <TabsTrigger value="npm" className="px-4">NPM</TabsTrigger>
                    <TabsTrigger value="pypi" className="px-4">PyPI</TabsTrigger>
                    <TabsTrigger value="recommended" className="px-4">Recommended</TabsTrigger>
                </TabsList>

                <TabsContent value="all" className="mt-6">
                    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
                        {services.map(service => (
                            <ServiceCard key={service.id} service={service} onViewDetails={viewServiceDetails} />
                        ))}
                    </div>
                </TabsContent>

                <TabsContent value="npm" className="mt-6">
                    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
                        {services
                            .filter(service => service.source === 'npm')
                            .map(service => (
                                <ServiceCard key={service.id} service={service} onViewDetails={viewServiceDetails} />
                            ))}
                    </div>
                </TabsContent>

                <TabsContent value="pypi" className="mt-6">
                    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
                        {services
                            .filter(service => service.source === 'pypi')
                            .map(service => (
                                <ServiceCard key={service.id} service={service} onViewDetails={viewServiceDetails} />
                            ))}
                        {services.filter(service => service.source === 'pypi').length === 0 && (
                            <div className="col-span-3 text-center py-8 text-muted-foreground">
                                <p>No PyPI services available yet.</p>
                            </div>
                        )}
                    </div>
                </TabsContent>

                <TabsContent value="recommended" className="mt-6">
                    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
                        {services
                            .filter(service => service.source === 'recommended')
                            .map(service => (
                                <ServiceCard key={service.id} service={service} onViewDetails={viewServiceDetails} />
                            ))}
                    </div>
                </TabsContent>
            </Tabs>
        </div>
    );
}

// 服务卡片组件
function ServiceCard({ service, onViewDetails }: { service: ServiceType, onViewDetails: (id: string) => void }) {
    return (
        <Card className="border-border shadow-sm hover:shadow transition-shadow duration-200">
            <CardHeader>
                <div className="flex items-center justify-between">
                    <div className="flex items-center">
                        <div className="bg-primary/10 p-2 rounded-md mr-3 flex-shrink-0">
                            {service.icon || <Package className="w-6 h-6 text-primary" />}
                        </div>
                        <div>
                            <CardTitle className="text-lg truncate max-w-[200px]" title={service.name}>{service.name}</CardTitle>
                            <CardDescription>
                                <span className={`inline-flex items-center text-xs`}>
                                    v{service.version} • {service.source}
                                </span>
                            </CardDescription>
                        </div>
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                <p className="text-sm text-muted-foreground mb-3 line-clamp-2 h-10" title={service.description}>{service.description}</p>
                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                    <div className="flex items-center gap-1">
                        <Download className="h-3.5 w-3.5" />
                        <span>{service.downloads.toLocaleString()}</span>
                    </div>
                    <div className="flex items-center gap-1">
                        <Star className="h-3.5 w-3.5" />
                        <span>{service.stars}</span>
                    </div>
                    <div className="truncate max-w-[100px]" title={`By ${service.author}`}>
                        <span>By {service.author}</span>
                    </div>
                </div>
            </CardContent>
            <CardFooter className="flex justify-between">
                <Button variant="outline" size="sm" onClick={() => onViewDetails(service.id)}>Details</Button>
                <Button size="sm">Install</Button>
            </CardFooter>
        </Card>
    );
} 