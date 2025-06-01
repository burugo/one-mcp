import React, { useState, useEffect, useRef } from 'react';
import { ServiceType, useMarketStore, InstallTask, InstallStatus, UninstallTask } from '@/store/marketStore'; // Adjust path if store is elsewhere
import { Button } from '@/components/ui/button'; // Assuming shadcn/ui button
import { Star, TrendingUp, Github, User, Package, CheckCircle2, Loader2 } from 'lucide-react';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'; // Import ConfirmDialog
import { toast } from '@/hooks/use-toast';

interface ServiceCardProps {
    service: ServiceType;
    onSelect: (serviceId: string) => void;
    onInstall: (serviceId: string) => void;
}

const ServiceCard: React.FC<ServiceCardProps> = ({ service, onSelect, onInstall }) => {
    const { uninstallService } = useMarketStore.getState(); // Get uninstallService action from store
    const installTask: InstallTask | undefined = useMarketStore(state => state.installTasks[service.id]);
    const uninstallTask: UninstallTask | undefined = useMarketStore(state => state.uninstallTasks[service.id]);

    const isEffectivelyInstalled = (service.isInstalled || installTask?.status === 'success') && uninstallTask?.status !== 'uninstalling';
    const isInstalling = installTask?.status === 'installing';
    const isUninstalling = uninstallTask?.status === 'uninstalling';

    const [showSuccessAnimation, setShowSuccessAnimation] = useState(false);
    const [showInstallingAnimation, setShowInstallingAnimation] = useState(false);
    const [showUninstallSuccessAnimation, setShowUninstallSuccessAnimation] = useState(false);
    const previousInstallStatusRef = useRef<InstallStatus | undefined>(undefined);
    const previousUninstallStatusRef = useRef<string | undefined>(undefined);

    const [isConfirmUninstallOpen, setIsConfirmUninstallOpen] = useState(false);

    useEffect(() => {
        const currentStatus = installTask?.status;

        if (currentStatus === 'installing') {
            setShowInstallingAnimation(true);
            setShowSuccessAnimation(false);
        } else if (currentStatus === 'success' && previousInstallStatusRef.current === 'installing') {
            setShowInstallingAnimation(false);
            setShowSuccessAnimation(true);
            const timer = setTimeout(() => {
                setShowSuccessAnimation(false);
            }, 2000);
            return () => clearTimeout(timer);
        } else if (currentStatus === 'idle' || currentStatus === 'error') {
            setShowInstallingAnimation(false);
            setShowSuccessAnimation(false);
        }

        previousInstallStatusRef.current = currentStatus;
    }, [installTask?.status]);

    useEffect(() => {
        const currentStatus = uninstallTask?.status;

        if (currentStatus === 'uninstalling') {
            setShowUninstallSuccessAnimation(false);
        } else if (currentStatus === 'idle' && previousUninstallStatusRef.current === 'uninstalling') {
            setShowUninstallSuccessAnimation(true);
            const timer = setTimeout(() => {
                setShowUninstallSuccessAnimation(false);
            }, 2000);
            return () => clearTimeout(timer);
        }

        previousUninstallStatusRef.current = currentStatus;
    }, [uninstallTask?.status]);

    const handleSelect = () => {
        onSelect(service.id);
    };

    const handleInstall = () => {
        // TODO: Trigger environment variable modal if needed before calling onInstall
        onInstall(service.id);
    };

    const handleUninstallClick = () => {
        setIsConfirmUninstallOpen(true);
    };

    const executeActualUninstall = async () => {
        if (!service || typeof service.installed_service_id !== 'number') {
            toast({
                variant: "destructive",
                title: "Cannot Uninstall",
                description: "Service data is incomplete or numeric service ID is missing for uninstall."
            });
            setIsConfirmUninstallOpen(false); // Close the dialog
            return;
        }
        try {
            await uninstallService(service.installed_service_id); // Use numeric ID
            toast({
                title: "Uninstall Complete",
                description: `${service.name} has been successfully uninstalled.`
            });
        } catch (error: any) {
            toast({
                variant: "destructive",
                title: "Uninstall Failed",
                description: error.message || "Failed to uninstall service."
            });
        }
    };

    const getAuthorDisplay = () => {
        if (service.author && typeof service.author === 'string' && service.author !== 'Unknown Author') {
            return service.author;
        } else if (service.author && typeof service.author === 'object' && service.author.name) {
            return service.author.name;
        }
        // Fallback for author if it's an object without a name, or if homepage is preferred for github owner
        if (service.homepage && service.homepage.includes('github.com')) { // Changed homepageUrl to homepage
            try {
                const url = new URL(service.homepage); // Changed homepageUrl to homepage
                const pathParts = url.pathname.split('/').filter(part => part.length > 0);
                if (pathParts.length > 0) {
                    return pathParts[0]; // GitHub owner/org
                }
            } catch (e) {
                // Fall through to default if URL parsing fails
            }
        }
        return 'Unknown Source'; // Default if no specific author or GitHub owner
    };

    const authorDisplay = getAuthorDisplay();
    const isGithub = !!(service.homepage && service.homepage.includes('github.com')); // Changed homepageUrl to homepage

    return (
        <div className="relative bg-card border border-border rounded-lg p-4 flex flex-col h-full shadow-sm hover:shadow-md transition-shadow duration-200 group">
            <div className="flex items-start gap-3 mb-2">
                <div className="bg-muted p-2 rounded-md">
                    <Package size={24} className="text-primary" />
                </div>
                <div>
                    <h3 className="text-lg font-semibold group-hover:text-primary transition-colors duration-200 truncate" title={service.name}>
                        <a href={`https://www.npmjs.com/package/${service.name}`} target="_blank" rel="noopener noreferrer" className="hover:underline">
                            {service.name}
                        </a>
                    </h3>
                    <p className="text-xs text-muted-foreground">
                        v{service.version} &bull; {service.source}
                    </p>
                </div>
                {service.homepage && ( // Changed homepageUrl to homepage
                    <a
                        href={service.homepage} // Changed homepageUrl to homepage
                        target="_blank"
                        rel="noopener noreferrer"
                        className="ml-auto text-muted-foreground hover:text-primary transition-colors"
                        title="Visit Homepage"
                    >
                        {service.homepage.includes('github.com') ? <Github size={20} /> : <Package size={20} />}
                    </a>
                )}
            </div>

            <p className="text-sm text-muted-foreground mt-1 mb-3 line-clamp-2 flex-grow" title={service.description}>
                {service.description || 'No description available.'}
            </p>

            <div className="mb-3 flex items-center gap-4 text-xs text-muted-foreground">
                {/* GitHub Stars Display (仅主页为 GitHub 且 stars>0 时显示) */}
                {isGithub && typeof service.stars === 'number' && service.stars > 0 && (
                    <div className="flex items-center gap-1" title={`${service.stars} GitHub Stars`}>
                        <Star size={14} className="text-yellow-400 fill-yellow-400" />
                        <span>{service.stars.toLocaleString()}</span>
                    </div>
                )}

                {/* npm Score Display (有值就显示) */}
                {typeof service.score === 'number' && ( // Changed npmScore to score
                    <div className="flex items-center gap-1" title={`${service.score} Score`}>
                        <TrendingUp size={14} className="text-blue-500" />
                        <span>{service.score.toLocaleString()}</span>
                    </div>
                )}

                {/* Author / Source Display */}
                <div className="flex items-center gap-1 truncate" title={`Author/Source: ${authorDisplay}`}>
                    <User size={14} />
                    <span className="truncate">{authorDisplay}</span>
                </div>
            </div>

            <div className="mt-auto pt-3 flex gap-2 border-t border-border/50">
                <Button variant="outline" size="sm" className="w-full" onClick={handleSelect}>
                    Details
                </Button>
                {isEffectivelyInstalled ? (
                    <Button
                        variant="destructive"
                        size="sm"
                        className="w-full"
                        onClick={handleUninstallClick}
                        disabled={isUninstalling}
                    >
                        {isUninstalling ? 'Uninstalling...' : 'Uninstall'}
                    </Button>
                ) : (
                    <Button
                        size="sm"
                        className="w-full"
                        onClick={handleInstall}
                        disabled={isInstalling || isUninstalling}
                    >
                        {isInstalling
                            ? 'Installing...'
                            : (service.isInstalled && uninstallTask?.status !== 'uninstalling')
                                ? 'Install'
                                : 'Install'}
                    </Button>
                )}
            </div>

            {/* Installing Animation Overlay */}
            {showInstallingAnimation && (
                <div className="absolute inset-0 flex items-center justify-center bg-card/75 dark:bg-black/75 rounded-lg z-10 pointer-events-none">
                    <Loader2 size={60} className="text-primary animate-spin" />
                </div>
            )}

            {/* Success Animation Overlay */}
            {(showSuccessAnimation || showUninstallSuccessAnimation) && (
                <div className="absolute inset-0 flex items-center justify-center bg-card/75 dark:bg-black/75 rounded-lg z-20 pointer-events-none">
                    <CheckCircle2
                        size={64}
                        className={`${showUninstallSuccessAnimation ? 'text-red-500' : 'text-green-500'} animate-success-tick`}
                    />
                </div>
            )}

            {/* Uninstall Confirmation Dialog */}
            <ConfirmDialog
                isOpen={isConfirmUninstallOpen}
                onOpenChange={setIsConfirmUninstallOpen}
                title="Confirm Uninstallation"
                description={`Are you sure you want to uninstall "${service.name}"? This action cannot be undone.`}
                confirmText="Uninstall"
                cancelText="Cancel"
                confirmButtonVariant="destructive"
                onConfirm={executeActualUninstall}
            />
        </div>
    );
};

export default ServiceCard; 