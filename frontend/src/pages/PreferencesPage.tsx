import { useEffect, useState } from 'react';
import api, { APIResponse } from '@/utils/api';
import { useServerAddressStore } from '@/hooks/useServerAddress';

export function PreferencesPage() {
    const serverAddress = useServerAddressStore(s => s.serverAddress);
    const setServerAddress = useServerAddressStore(s => s.setServerAddress);
    const fetchServerAddress = useServerAddressStore(s => s.fetchServerAddress);
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [message, setMessage] = useState('');

    useEffect(() => {
        setLoading(true);
        fetchServerAddress().finally(() => setLoading(false));
    }, [fetchServerAddress]);

    const handleSave = async () => {
        setSaving(true);
        setMessage('');
        const clean = serverAddress.replace(/\/$/, '');
        const res = await api.put('/option/', { key: 'ServerAddress', value: clean }) as APIResponse;
        if (res.success) {
            setMessage('保存成功');
            setServerAddress(clean); // 立即同步到全局
        } else {
            setMessage(res.message || '保存失败');
        }
        setSaving(false);
    };

    return (
        <div className="w-full space-y-8">
            <h2 className="text-3xl font-bold tracking-tight mb-8">Preferences</h2>
            <div className="max-w-lg space-y-4">
                <label className="block text-sm font-medium mb-1">服务器地址 (ServerAddress)</label>
                <input
                    type="text"
                    className="w-full border rounded px-3 py-2"
                    value={serverAddress}
                    onChange={e => setServerAddress(e.target.value)}
                    placeholder="如 http://localhost:3000"
                    disabled={loading || saving}
                />
                <button
                    className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-60"
                    onClick={handleSave}
                    disabled={loading || saving}
                >
                    {saving ? '保存中...' : '保存'}
                </button>
                {message && <div className="text-green-600 mt-2">{message}</div>}
            </div>
        </div>
    );
} 