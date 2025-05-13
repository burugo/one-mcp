import { useOutletContext } from 'react-router-dom';
import type { PageOutletContext } from '../App';

export function AnalyticsPage() {
    // const { setIsOpen } = useOutletContext<PageOutletContext>(); // Ready for future use
    useOutletContext<PageOutletContext>(); // Establish context connection

    return (
        <div className="w-full space-y-8">
            <h2 className="text-3xl font-bold tracking-tight mb-8">Analytics</h2>
            {/* ... analytics content will go here ... */}
            <p className="text-muted-foreground">Analytics page content is under construction.</p>
        </div>
    );
} 