import { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

import { useToast } from '@/hooks/use-toast';
import { ChevronLeft, Package, Star, Download, CheckCircle, XCircle, AlertCircle, ExternalLink } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { useMarketStore } from '@/store/marketStore';
import EnvVarInputModal from './EnvVarInputModal';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { useTranslation } from 'react-i18next';

export function ServiceDetails({ onBack }: { onBack: () => void }) {
    const { t } = useTranslation();
    const { toast } = useToast();
    const {
        selectedService,
        isLoadingDetails,
        installTasks,
        installService,
        uninstallService,
        updateInstallStatus,
        fetchServiceDetails
    } = useMarketStore();

    // State for installation log dialog
    const [showInstallDialog, setShowInstallDialog] = useState(false);

    // State for EnvVarInputModal (similar to ServiceMarketplace)
    const [envModalVisible, setEnvModalVisible] = useState(false);
    const [missingVars, setMissingVars] = useState<string[]>([]);
    const [pendingServiceId, setPendingServiceId] = useState<string | null>(null);
    const [currentEnvVars, setCurrentEnvVars] = useState<Record<string, string>>({});
    // Reset modal states if selectedService changes
    useEffect(() => {
        setEnvModalVisible(false);
        setMissingVars([]);
        setPendingServiceId(null);
        setCurrentEnvVars({});
        setShowInstallDialog(false); // Also reset log dialog
    }, [selectedService?.id]);

