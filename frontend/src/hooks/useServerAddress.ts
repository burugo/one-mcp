import { useEffect } from 'react';
import { create } from 'zustand';
import api from '@/utils/api';

interface ServerAddressState {
    serverAddress: string;
    setServerAddress: (addr: string) => void;
    fetchServerAddress: () => Promise<void>;
}

export const useServerAddressStore = create<ServerAddressState>((set) => ({
    serverAddress: '',
    setServerAddress: (addr) => set({ serverAddress: addr }),
    fetchServerAddress: async () => {
        const res = await api.get('/option/');
        if (res.success && Array.isArray(res.data)) {
            const found = res.data.find((item: any) => item.key === 'ServerAddress');
            if (found) set({ serverAddress: found.value });
        }
    },
}));

export function useServerAddress() {
    const serverAddress = useServerAddressStore(s => s.serverAddress);
    const fetchServerAddress = useServerAddressStore(s => s.fetchServerAddress);
    useEffect(() => { fetchServerAddress(); }, [fetchServerAddress]);
    return serverAddress;
} 