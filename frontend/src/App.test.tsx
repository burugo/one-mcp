import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AppContent } from './App';

describe('AppContent', () => {
    it('renders the main application content without crashing', () => {
        render(
            <MemoryRouter>
                <AppContent />
            </MemoryRouter>
        );
        // Example assertion (can be refined based on AppContent's initial route)
        // For instance, if /dashboard is the initial route, check for Dashboard heading:
        // expect(screen.getByRole('heading', { name: /Dashboard/i })).toBeInTheDocument();
    });
}); 