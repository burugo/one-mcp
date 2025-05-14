import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AppContent } from './App';
import { AuthProvider } from './contexts/AuthContext';

describe('AppContent', () => {
    it('renders the main application content without crashing', () => {
        render(
            <MemoryRouter initialEntries={['/']}>
                <AuthProvider>
                    <AppContent />
                </AuthProvider>
            </MemoryRouter>
        );
        // Example: Check for something that should be visible when auth is loading or on a public route
        // This assertion might need adjustment based on initial state from AuthProvider
        // expect(screen.getByText(/Loading application.../i)).toBeInTheDocument(); 
        // Or, if a public part of AppLayout is rendered immediately:
        expect(screen.getAllByText(/One MCP/i).length).toBeGreaterThanOrEqual(1); // Check for at least one app title occurrence
    });
}); 