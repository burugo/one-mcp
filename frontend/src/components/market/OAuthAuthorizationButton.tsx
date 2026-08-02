import { useState } from 'react';
import { KeyRound, Loader2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import type { ServiceType } from '@/store/marketStore';
import api, { APIResponse } from '@/utils/api';

interface OAuthAuthorizationButtonProps {
  service: Pick<ServiceType, 'id' | 'name' | 'display_name' | 'oauth_enabled' | 'oauth_configured' | 'oauth_auth_status'>;
}

export function OAuthAuthorizationButton({ service }: OAuthAuthorizationButtonProps) {
  const { t } = useTranslation();
  const { toast } = useToast();
  const [isAuthorizing, setIsAuthorizing] = useState(false);

  const oauthConfigured = service.oauth_enabled || service.oauth_configured;
  if (!oauthConfigured || (service.oauth_enabled && service.oauth_auth_status === 'authorized')) {
    return null;
  }

  const actionLabel = t('services.oauthAuthorizeInBrowser');

  const handleAuthorize = async () => {
    if (isAuthorizing) return;

    setIsAuthorizing(true);
    try {
      const response = await api.post(`/mcp_services/${service.id}/oauth/authorize`) as APIResponse<{ authorization_url: string }>;
      if (!response.success || !response.data?.authorization_url) {
        throw new Error(response.message || t('services.oauthAuthorizationUrlMissing'));
      }
      window.open(response.data.authorization_url, '_self', 'noopener,noreferrer');
    } catch (error) {
      toast({
        title: t('services.oauthAuthorizationFailed'),
        description: error instanceof Error ? error.message : t('common.unknownError'),
        variant: 'destructive',
      });
      setIsAuthorizing(false);
    }
  };

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className="h-7 w-7 shrink-0 rounded-full text-amber-500 hover:bg-amber-500/10 hover:text-amber-600"
      disabled={isAuthorizing}
      aria-label={actionLabel}
      title={actionLabel}
      onClick={handleAuthorize}
    >
      {isAuthorizing ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />}
    </Button>
  );
}
