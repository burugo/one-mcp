import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ServicesPage } from './ServicesPage';
import { useMarketStore } from '@/store/marketStore';
import { createMockService } from '@/__tests__/utils/test-utils';
import api, { toastEmitter } from '@/utils/api';

vi.mock('@/utils/api', () => ({
    default: { get: vi.fn(), post: vi.fn() },
    toastEmitter: { emit: vi.fn(), on: vi.fn(), off: vi.fn() },
}));
vi.mock('@/contexts/AuthContext', () => ({
    useAuth: () => ({ currentUser: { id: 1, username: 'admin', role: 10 }, updateUserInfo: vi.fn() }),
}));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));

describe('service health refresh', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(localStorage.getItem).mockReturnValue(null);
        useMarketStore.setState({ installedServices: [], isSearching: false });
    });
    afterEach(() => vi.mocked(localStorage.getItem).mockReset());

    it.each([
        ['grid', 3], ['list', 3], ['grid', 0], ['list', 0],
    ] as const)('shows refreshed policy counts in %s view with %i enabled tools', async (view, enabledCount) => {
        vi.mocked(localStorage.getItem).mockImplementation(key => key === 'services_view_mode' ? view : null);
        const dormant = createMockService({ health_status: 'unknown', total_tool_count: 0, enabled_tool_count: 0 });
        const checked = { ...dormant, health_status: 'healthy', tool_count: 4, total_tool_count: 4, enabled_tool_count: enabledCount };
        let healthChecked = false;
        vi.mocked(api.get).mockImplementation(async (path) => {
            if (path === '/mcp_market/installed') return { success: true, data: [healthChecked ? checked : dormant] };
            throw new Error(`Unexpected GET ${path}`);
        });
        vi.mocked(api.post).mockImplementation(async (path) => {
            if (path !== '/mcp_services/1/health/check') throw new Error(`Unexpected POST ${path}`);
            healthChecked = true;
            return { success: true, data: { health_status: 'healthy', health_details: { tool_count: 4 }, last_checked: '2026-09-06T12:00:00Z' } };
        });

        render(<MemoryRouter><ServicesPage /></MemoryRouter>);
        await screen.findByTitle('services.refreshHealthStatus');
        const badge = `${enabledCount}/4${view === 'list' ? ' tools' : ''}`;
        expect(screen.queryByText(badge)).not.toBeInTheDocument();
        fireEvent.click(screen.getByTitle('services.refreshHealthStatus'));
        await waitFor(() => expect(screen.getByText(badge)).toBeInTheDocument());
        expect(useMarketStore.getState().installedServices[0].enabled).toBe(true);
        expect(api.post).toHaveBeenCalledTimes(1);
    });

    it('does not enable a disabled service or invent tool counts when its check is rejected', async () => {
        const disabled = createMockService({ enabled: false, health_status: 'unknown', total_tool_count: 0, enabled_tool_count: 0 });
        vi.mocked(api.get).mockResolvedValue({ success: true, data: [disabled] });
        vi.mocked(api.post).mockResolvedValue({ success: false, message: 'service is disabled or uninstalled' });
        render(<MemoryRouter><ServicesPage /></MemoryRouter>);
        await screen.findByTitle('services.refreshHealthStatus');
        fireEvent.click(screen.getByTitle('services.refreshHealthStatus'));
        await waitFor(() => expect(toastEmitter.emit).toHaveBeenCalledWith(expect.objectContaining({
            variant: 'destructive', description: 'service is disabled or uninstalled',
        })));
        expect(useMarketStore.getState().installedServices[0]).toMatchObject({ enabled: false, total_tool_count: 0 });
        expect(api.post).toHaveBeenCalledTimes(1);
    });
});
