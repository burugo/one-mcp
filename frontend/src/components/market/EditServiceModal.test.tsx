import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import EditServiceModal from './EditServiceModal';

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
  toast: vi.fn(),
}));

vi.mock('@/utils/api', () => ({
  default: {
    get: mocks.get,
    put: mocks.put,
    post: mocks.post,
    delete: mocks.delete,
  },
}));

vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: mocks.toast }),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => ({
      'editServiceModal.oauth.title': 'OAuth',
      'editServiceModal.oauth.sharedDescription': 'One administrator authorization is shared globally.',
      'editServiceModal.oauth.status': 'Status',
      'editServiceModal.oauth.callback': 'Callback',
      'editServiceModal.oauth.login': 'OAuth Login',
      'editServiceModal.oauth.advancedSettings': 'Advanced OAuth settings',
      'editServiceModal.oauth.advancedDescription': 'Only fill these fields when required.',
      'editServiceModal.oauth.clientId': 'Client ID',
      'editServiceModal.oauth.clientSecret': 'Client Secret',
      'editServiceModal.oauth.scopes': 'Scopes',
      'editServiceModal.oauth.scopesPlaceholder': 'read write',
      'editServiceModal.oauth.statuses.auth_required': 'Authorization required',
      'editServiceModal.oauth.statuses.authorized': 'Authorized',
      'editServiceModal.oauth.disableConfirmTitle': 'Disable OAuth?',
      'editServiceModal.oauth.disableConfirmDescription': 'This removes the shared OAuth credentials and token.',
      'editServiceModal.oauth.disableConfirmAction': 'Disable OAuth',
      'editServiceModal.oauth.disableCancelAction': 'Keep OAuth enabled',
    }[key] || key),
  }),
}));

describe('EditServiceModal OAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(window, 'open').mockImplementation(() => null);
    mocks.get.mockResolvedValue({
      success: true,
      data: { status: 'auth_required', scopes: '' },
    });
    mocks.put.mockResolvedValue({ success: true, data: { status: 'auth_required' } });
    mocks.post.mockResolvedValue({
      success: true,
      data: { authorization_url: 'https://accounts.example.test/authorize' },
    });
    mocks.delete.mockResolvedValue({ success: true });
  });

  it('uses DCR by default and hides optional OAuth fields under advanced settings', async () => {
    render(
      <EditServiceModal
        open
        onClose={vi.fn()}
        onUpdateService={vi.fn()}
        service={{
          id: '42',
          name: 'remote-service',
          display_name: 'Remote Service',
          description: '',
          version: '1.0.0',
          source: 'local',
          envVars: [],
          type: 'streamableHttp',
          command: 'https://mcp.example.test/mcp',
          oauth_enabled: true,
          oauth_auth_status: 'auth_required',
        }}
      />,
    );

    expect(screen.queryByLabelText('Client ID')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Client Secret')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Scopes')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'OAuth Login' }));

    await waitFor(() => {
      expect(mocks.put).toHaveBeenCalledWith('/mcp_services/42/oauth', {
        client_id: '',
        client_secret: '',
        scopes: '',
      });
      expect(mocks.post).toHaveBeenCalledWith('/mcp_services/42/oauth/authorize');
      expect(window.open).toHaveBeenCalledWith('https://accounts.example.test/authorize', '_self', 'noopener,noreferrer');
    });

    fireEvent.click(screen.getByRole('button', { name: 'Advanced OAuth settings' }));
    expect(screen.getByLabelText('Client ID')).toBeInTheDocument();
    expect(screen.getByLabelText('Client Secret')).toBeInTheDocument();
    expect(screen.getByLabelText('Scopes')).toBeInTheDocument();
  });

  it('requires confirmation before disabling OAuth', async () => {
    mocks.get.mockResolvedValue({
      success: true,
      data: { status: 'authorized', scopes: 'mcp:read' },
    });
    render(
      <EditServiceModal
        open
        onClose={vi.fn()}
        onUpdateService={vi.fn()}
        service={{
          id: '42',
          name: 'remote-service',
          display_name: 'Remote Service',
          description: '',
          version: '1.0.0',
          source: 'local',
          envVars: [],
          type: 'streamableHttp',
          command: 'https://mcp.example.test/mcp',
          oauth_enabled: true,
          oauth_auth_status: 'authorized',
        }}
      />,
    );

    const oauthCheckbox = screen.getByLabelText('OAuth');
    fireEvent.click(oauthCheckbox);

    expect(mocks.delete).not.toHaveBeenCalled();
    expect(oauthCheckbox).toBeChecked();
    expect(screen.getByRole('alertdialog', { name: 'Disable OAuth?' })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Keep OAuth enabled' }));
    expect(mocks.delete).not.toHaveBeenCalled();
    expect(oauthCheckbox).toBeChecked();

    fireEvent.click(oauthCheckbox);
    fireEvent.click(screen.getByRole('button', { name: 'Disable OAuth' }));

    await waitFor(() => {
      expect(mocks.delete).toHaveBeenCalledWith('/mcp_services/42/oauth');
      expect(oauthCheckbox).not.toBeChecked();
    });
  });

  it('keeps a disabled OAuth configuration available without checking the checkbox', async () => {
    render(
      <EditServiceModal
        open
        onClose={vi.fn()}
        onUpdateService={vi.fn()}
        service={{
          id: '42',
          name: 'remote-service',
          display_name: 'Remote Service',
          description: '',
          version: '1.0.0',
          source: 'local',
          envVars: [],
          type: 'streamableHttp',
          command: 'https://mcp.example.test/mcp',
          oauth_enabled: false,
          oauth_configured: true,
          oauth_auth_status: 'auth_required',
        }}
      />,
    );

    expect(screen.getByLabelText('OAuth')).not.toBeChecked();
    await waitFor(() => {
      expect(mocks.get).toHaveBeenCalledWith('/mcp_services/42/oauth');
    });
  });
});
