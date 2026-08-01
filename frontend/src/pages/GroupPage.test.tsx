import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { render } from '@/__tests__/utils/test-utils';
import { GroupPage } from './GroupPage';

const mocks = vi.hoisted(() => ({
  getGroups: vi.fn(),
  getServices: vi.fn(),
}));

vi.mock('@/utils/api', () => ({
  GroupService: {
    getAll: mocks.getGroups,
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
  default: {
    get: mocks.getServices,
  },
}));

vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}));

vi.mock('@/hooks/useServerAddress', () => ({
  useServerAddress: () => 'http://localhost:3000',
}));

vi.mock('@/contexts/AuthContext', () => ({
  useAuth: () => ({
    currentUser: null,
    updateUserInfo: vi.fn(),
  }),
}));

describe('GroupPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getGroups.mockResolvedValue({
      success: true,
      data: [{
        id: 1,
        name: 'search-group',
        display_name: 'Search Group',
        description: 'This group contains the following MCP services:\n- exa: 用于搜索',
        service_ids_json: '[1]',
        enabled: true,
      }],
    });
    mocks.getServices.mockResolvedValue({
      success: true,
      data: [{
        id: 1,
        name: 'exa',
        display_name: 'Exa',
        description: '用于搜索',
        enabled: true,
      }],
    });
  });

  it('renders the current service description in the group editor', async () => {
    render(<GroupPage />);

    await waitFor(() => {
      expect(screen.getByText('Search Group')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTitle('common.edit'));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('用于搜索')).toBeInTheDocument();
  });
});
