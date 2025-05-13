import { useOutletContext } from 'react-router-dom';
import type { PageOutletContext } from '../App';

export function ProfilePage() {
    // const { setIsOpen } = useOutletContext<PageOutletContext>(); // Ready for future use
    useOutletContext<PageOutletContext>(); // Establish context connection

    return (
        <div className="w-full space-y-8">
            <h2 className="text-3xl font-bold tracking-tight mb-8">User Profile</h2>
            {/* ... profile content will go here ... */}
            <p className="text-muted-foreground">User Profile page content is under construction.</p>
        </div>
    );
} 