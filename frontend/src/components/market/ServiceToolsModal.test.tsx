import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ServiceToolsModal from './ServiceToolsModal';

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  toast: vi.fn(),
}));

vi.mock('@/utils/api', () => ({
  default: { get: mocks.get, put: mocks.put },
}));

vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: mocks.toast }),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => ({
      'serviceTools.toolsList': 'Tools',
      'serviceTools.availableToolsDescription': 'Discovered tools',
      'serviceTools.noToolsAvailable': 'No tools',
      'serviceTools.description': 'Description',
      'serviceTools.parameters': 'Parameters',
      'serviceTools.parameterName': 'Name',
      'serviceTools.parameterType': 'Type',
      'serviceTools.parameterDescription': 'Description',
      'serviceTools.enableTool': 'Enable',
      'serviceTools.enabled': 'Enabled',
      'serviceTools.disabled': 'Disabled',
      'serviceTools.updateFailed': 'Failed to update tool',
      'common.loading': 'Loading',
    }[key] || key),
  }),
}));

describe('ServiceToolsModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.get.mockResolvedValue({
      success: true,
      data: {
        tools: [
          { name: 'allowed-tool', description: 'Allowed details', enabled: true },
          { name: 'blocked-tool', description: 'Blocked details', enabled: false },
        ],
        total_tool_count: 2,
        enabled_tool_count: 1,
        disabled_tool_count: 1,
      },
    });
    mocks.put.mockResolvedValue({ success: true });
  });

  it('keeps tool expansion separate from the administrator switch', async () => {
    render(
      <ServiceToolsModal
        serviceId="7"
        serviceName="Example MCP"
        isOpen
        canManageTools
        onClose={vi.fn()}
      />,
    );

    expect(await screen.findByText('1/2')).toBeInTheDocument();
    const blockedTrigger = screen.getByRole('button', { name: /blocked-tool/i });
    fireEvent.click(blockedTrigger);
    expect(blockedTrigger).toHaveAttribute('data-state', 'open');

    const toggle = screen.getByRole('switch', { name: 'Enable blocked-tool' });
    fireEvent.click(toggle);
    await waitFor(() => {
      expect(mocks.put).toHaveBeenCalledWith('/mcp_services/7/tools/policy', {
        tool_name: 'blocked-tool',
        enabled: true,
      });
    });
    expect(blockedTrigger).toHaveAttribute('data-state', 'open');
    expect(toggle).toBeChecked();
    expect(screen.getByText('2/2')).toBeInTheDocument();
  });

  it('shows read-only policy state to non-admin users', async () => {
    render(
      <ServiceToolsModal
        serviceId="7"
        serviceName="Example MCP"
        isOpen
        canManageTools={false}
        onClose={vi.fn()}
      />,
    );

    expect(await screen.findByText('Disabled')).toBeInTheDocument();
    expect(screen.queryByRole('switch')).not.toBeInTheDocument();
  });

  it('keeps the previous state and reports an error when an update fails', async () => {
    mocks.put.mockRejectedValue(new Error('network failed'));
    render(
      <ServiceToolsModal
        serviceId="7"
        serviceName="Example MCP"
        isOpen
        canManageTools
        onClose={vi.fn()}
      />,
    );

    const toggle = await screen.findByRole('switch', { name: 'Enable blocked-tool' });
    fireEvent.click(toggle);
    await waitFor(() => expect(mocks.toast).toHaveBeenCalled());
    expect(toggle).not.toBeChecked();
    expect(screen.getByText('1/2')).toBeInTheDocument();
  });
});
