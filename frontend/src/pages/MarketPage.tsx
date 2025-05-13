import { useState } from 'react';
import { ServiceMarketplace } from '../components/market/ServiceMarketplace';
import { ServiceDetails } from '../components/market/ServiceDetails';

export function MarketPage() {
    const [selectedServiceId, setSelectedServiceId] = useState<string | null>(null);

    // 查看服务详情
    const viewServiceDetails = (serviceId: string) => {
        setSelectedServiceId(serviceId);
    };

    // 返回市场页面
    const goBackToMarketplace = () => {
        setSelectedServiceId(null);
    };

    // 根据是否选择了服务显示不同的组件
    return (
        <div className="w-full">
            {selectedServiceId ? (
                <ServiceDetails serviceId={selectedServiceId} onBack={goBackToMarketplace} />
            ) : (
                <ServiceMarketplace onSelectService={viewServiceDetails} />
            )}
        </div>
    );
} 