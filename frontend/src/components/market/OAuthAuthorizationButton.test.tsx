import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { OAuthAuthorizationButton } from './OAuthAuthorizationButton';

const mocks = vi.hoisted(() => ({
  post: vi.fn(),
  toast: vi.fn(),
}));

vi.mock('@/utils/api', () => ({
  default: { post: mocks.post },
}));

vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({ toast: mocks.toast }),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => ({
      'services.oauthAuthorized': 'OAuth authorized',
      'services.oauthAuthorizationRequired': 'OAuth authorization required',
      'services.oauthAuthorizeInBrowser': 'Authorize in browser',
      'services.oauthReauthorize': 'Reauthorize in browser',
      'services.oauthAuthorizationFailed': 'OAuth authorization failed',
      'services.oauthAuthorizationUrlMissing': 'The authorization URL was not returned',
    }[key] || key),
  }),
}));

describe('OAuthAuthorizationButton', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(window, 'open').mockImplementation(() => null);
  });

  it('starts OAuth authorization from the service list and redirects the browser', async () => {
    mocks.post.mockResolvedValue({
      success: true,
      data: { authorization_url: 'https://accounts.example.test/authorize' },
    });

    render(
      <OAuthAuthorizationButton
        service={{
          id: '42',
          name: 'remote-service',
          display_name: 'Remote Service',
          oauth_enabled: true,
          oauth_auth_status: 'auth_required',
        }}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Authorize in browser' }));

    await waitFor(() => {
      expect(mocks.post).toHaveBeenCalledWith('/mcp_services/42/oauth/authorize');
      expect(window.open).toHaveBeenCalledWith('https://accounts.example.test/authorize', '_self', 'noopener,noreferrer');
    });
  });

  it('allows a disabled configured service to start OAuth authorization', async () => {
    mocks.post.mockResolvedValue({
      success: true,
      data: { authorization_url: 'https://accounts.example.test/authorize' },
    });

    render(
      <OAuthAuthorizationButton
        service={{
          id: '42',
          name: 'remote-service',
          display_name: 'Remote Service',
          oauth_enabled: false,
          oauth_configured: true,
          oauth_auth_status: 'auth_required',
        }}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Authorize in browser' }));

    await waitFor(() => {
      expect(mocks.post).toHaveBeenCalledWith('/mcp_services/42/oauth/authorize');
      expect(window.open).toHaveBeenCalledWith('https://accounts.example.test/authorize', '_self', 'noopener,noreferrer');
    });
  });

  it('does not render an authorization icon after OAuth is authorized', () => {
    render(
      <OAuthAuthorizationButton
        service={{
          id: '42',
          name: 'remote-service',
          display_name: 'Remote Service',
          oauth_enabled: true,
          oauth_configured: true,
          oauth_auth_status: 'authorized',
        }}
      />,
    );

    expect(screen.queryByRole('button', { name: 'Authorize in browser' })).not.toBeInTheDocument();
  });
});
