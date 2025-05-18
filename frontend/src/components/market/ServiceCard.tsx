import React from 'react';
import { ServiceType, useMarketStore, InstallTask } from '@/store/marketStore'; // Adjust path if store is elsewhere
import { Button } from '@/components/ui/button'; // Assuming shadcn/ui button
import { Star, TrendingUp, Github, User, Package } from 'lucide-react';

interface ServiceCardProps {
    service: ServiceType;
    onSelect: (serviceId: string) => void;
    onInstall: (serviceId: string) => void;
}

const ServiceCard: React.FC<ServiceCardProps> = ({ service, onSelect, onInstall }) => {
    const installTask: InstallTask | undefined = useMarketStore(state => state.installTasks[service.id]);
    const isInstalled = service.isInstalled || (installTask?.status === 'success');

    const handleSelect = () => {
        onSelect(service.id);
    };

    const handleInstall = () => {
        // TODO: Trigger environment variable modal if needed before calling onInstall
        onInstall(service.id);
    };

    const getAuthorDisplay = () => {
        if (service.author && service.author !== 'Unknown Author') {
            return service.author;
        }
        if (service.homepageUrl && service.homepageUrl.includes('github.com')) {
            try {
                const url = new URL(service.homepageUrl);
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
    const isGithub = !!(service.homepageUrl && service.homepageUrl.includes('github.com'));

    return (
        <div className="bg-card border border-border rounded-lg p-4 flex flex-col h-full shadow-sm hover:shadow-md transition-shadow duration-200 group">
            <div className="flex items-start gap-3 mb-2">
                <div className="bg-muted p-2 rounded-md">
                    <Package size={24} className="text-primary" />
                </div>
                <div>
                    <h3 className="text-lg font-semibold group-hover:text-primary transition-colors duration-200 truncate" title={service.name}>
                        {service.name}
                    </h3>
                    <p className="text-xs text-muted-foreground">
                        v{service.version} &bull; {service.source}
                    </p>
                </div>
                {service.homepageUrl && (
                    <a
                        href={service.homepageUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="ml-auto text-muted-foreground hover:text-primary transition-colors"
                        title="Visit Homepage"
                    >
                        {service.homepageUrl.includes('github.com') ? <Github size={20} /> : <Package size={20} />}
                    </a>
                )}
            </div>

            <p className="text-sm text-muted-foreground mt-1 mb-3 line-clamp-2 flex-grow" title={service.description}>
                {service.description || 'No description available.'}
            </p>

            <div className="mb-3 flex items-center gap-4 text-xs text-muted-foreground">
                {/* GitHub Stars Display (仅主页为 GitHub 且 stars>0 时显示) */}
                {isGithub && Number.isFinite(service.stars) && service.stars > 0 && (
                    <div className="flex items-center gap-1" title={`${service.stars} GitHub Stars`}>
                        <Star size={14} className="text-yellow-400 fill-yellow-400" />
                        <span>{service.stars.toLocaleString()}</span>
                    </div>
                )}

                {/* npm Score Display (有值就显示) */}
                {Number.isFinite(service.npmScore) && (
                    <div className="flex items-center gap-1" title={`${service.npmScore} npm Score`}>
                        <TrendingUp size={14} className="text-blue-500" />
                        <span>{service.npmScore.toLocaleString()}</span>
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
                <Button
                    size="sm"
                    className="w-full"
                    onClick={handleInstall}
                    disabled={isInstalled || (installTask && installTask.status === 'installing')}
                >
                    {installTask && installTask.status === 'installing'
                        ? 'Installing...'
                        : isInstalled
                            ? 'Installed'
                            : 'Install'}
                </Button>
            </div>
        </div>
    );
};

export default ServiceCard; 