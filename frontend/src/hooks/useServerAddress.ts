import { useEffect } from 'react';
import { create } from 'zustand';
import api, { APIResponse } from '@/utils/api';

interface ServerAddressState {
    serverAddress: string;
    setServerAddress: (addr: string) => void;
    fetchServerAddress: () => Promise<void>;
}

export const useServerAddressStore = create<ServerAddressState>((set) => ({
    serverAddress: '',
    setServerAddress: (addr) => set({ serverAddress: addr }),
    fetchServerAddress: async () => {
        // 首先尝试从 sessionStorage 获取
        const cachedAddress = sessionStorage.getItem('server_address');
        if (cachedAddress) {
            set({ serverAddress: cachedAddress });
            return;
        }

        // 如果 sessionStorage 没有，从 /status 接口获取
        try {
            const res = await api.get('/status') as APIResponse;
            if (res.success && res.data?.server_address) {
                const serverAddress = res.data.server_address;
                // 保存到 sessionStorage
                sessionStorage.setItem('server_address', serverAddress);
                set({ serverAddress: serverAddress });
            }
        } catch (error) {
            console.error('Failed to fetch server address from /status:', error);
            // 如果 /status 失败，回退到原来的 /option/ 接口
            try {
                const res = await api.get('/option/') as APIResponse;
                if (res.success && Array.isArray(res.data)) {
                    const found = res.data.find((item: any) => item.key === 'ServerAddress');
                    if (found) {
                        const serverAddress = found.value;
                        // 保存到 sessionStorage
                        sessionStorage.setItem('server_address', serverAddress);
                        set({ serverAddress: serverAddress });
                    }
                }
            } catch (fallbackError) {
                console.error('Failed to fetch server address from /option/:', fallbackError);
            }
        }
    },
}));

export function useServerAddress() {
    const serverAddress = useServerAddressStore(s => s.serverAddress);
    const fetchServerAddress = useServerAddressStore(s => s.fetchServerAddress);
    useEffect(() => { fetchServerAddress(); }, [fetchServerAddress]);
    return serverAddress;
} 