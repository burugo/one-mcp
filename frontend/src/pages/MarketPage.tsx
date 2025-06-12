import { useMarketStore } from '../store/marketStore';
import { ServiceMarketplace } from '../components/market/ServiceMarketplace';
import { ServiceDetails } from '../components/market/ServiceDetails';


export function MarketPage() {
    const {
        selectedService,
        selectService,
        clearSelectedService
    } = useMarketStore();

    // 查看服务详情
    const viewServiceDetails = (serviceId: string) => {
        selectService(serviceId);
    };

    // 返回市场页面
    const goBackToMarketplace = () => {
        clearSelectedService();
    };

    // 根据是否选择了服务显示不同的组件
    return (
        <div className="w-full">
            {selectedService ? (
                <ServiceDetails onBack={goBackToMarketplace} />
            ) : (
                <ServiceMarketplace onSelectService={viewServiceDetails} />
            )}
        </div>
    );
} 