import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import CustomServiceModal from './CustomServiceModal';

const mocks = vi.hoisted(() => ({
  post: vi.fn(),
  createService: vi.fn(),
}));

vi.mock('@/utils/api', () => ({
  default: { post: mocks.post },
}));

vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: vi.fn() }),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => ({
      'customServiceModal.form.serviceName': 'Service Name',
      'customServiceModal.form.serverUrl': 'Server URL',
      'customServiceModal.actions.createAndAuthorize': 'Create and OAuth Login',
      'customServiceModal.oauth.detected': 'OAuth detected',
      'customServiceModal.oauth.defaultScopes': 'Default scopes:',
    }[key] || key),
  }),
}));

describe('CustomServiceModal OAuth discovery', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.post.mockResolvedValue({
      success: true,
      data: {
        oauth_available: true,
        automatic_authorization_supported: true,
        authorization_server: 'https://accounts.example.test',
        protected_resource_metadata_url: 'https://mcp.example.test/.well-known/oauth-protected-resource/mcp',
        scopes: ['mcp:read', 'mcp:write'],
        dynamic_client_registration_supported: true,
        pkce_s256_supported: true,
      },
    });
  });

  it('discovers OAuth after the URL loses focus and offers create-and-authorize', async () => {
    render(
      <CustomServiceModal
        open
        onClose={vi.fn()}
        onCreateService={mocks.createService}
        autoFillEnv=""
        setAutoFillEnv={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText('Service Name'), { target: { value: 'Remote MCP' } });
    const urlInput = screen.getByLabelText('Server URL');
    fireEvent.change(urlInput, { target: { value: 'https://mcp.example.test/mcp' } });
    fireEvent.blur(urlInput);

    await waitFor(() => {
      expect(mocks.post).toHaveBeenCalledWith('/mcp_market/oauth/discover', {
        url: 'https://mcp.example.test/mcp',
        type: 'streamableHttp',
      });
      expect(screen.getByText('OAuth detected')).toBeInTheDocument();
      expect(screen.getByText('mcp:read mcp:write')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Create and OAuth Login' }));

    await waitFor(() => {
      expect(mocks.createService).toHaveBeenCalledWith(expect.objectContaining({
        oauth_enabled: true,
        oauth_scopes: 'mcp:read mcp:write',
        oauth_protected_resource_metadata_url: 'https://mcp.example.test/.well-known/oauth-protected-resource/mcp',
      }));
    });
  });
});
