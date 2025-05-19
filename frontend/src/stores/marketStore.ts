export const useMarketStore = create<MarketStore>((set, get) => ({
    restartService: async (serviceId: number) => {
        try {
            const response = await api.post(`/api/mcp_services/${serviceId}/restart`);
            if (response.data.code === 0) {
                // 重新获取已安装服务列表
                await get().fetchInstalledServices();
                return true;
            }
            return false;
        } catch (error) {
            console.error('Failed to restart service:', error);
            return false;
        }
    },
})); 