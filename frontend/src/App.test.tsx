import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AppContent } from './App';
import { AuthProvider } from './contexts/AuthContext';
import { createInstance } from 'i18next';
import { I18nextProvider } from 'react-i18next';

describe('AppContent', () => {
    afterEach(() => vi.mocked(localStorage.getItem).mockReset());
    it.each([false, true])('renders the main application after authentication initializes (signed in: %s)', async (signedIn) => {
        vi.mocked(localStorage.getItem).mockImplementation(key => {
            if (!signedIn) return null;
            if (key === 'token') return 'fixture-session';
            if (key === 'user') return JSON.stringify({ id: 1, username: 'fixture-user', display_name: 'Fixture User' });
            return null;
        });
        // Production initializes i18n in main.tsx, which this component test does not mount.
        const i18n = createInstance();
        await i18n.init({ lng: 'en', resources: { en: { translation: {} } } });
        render(
            <MemoryRouter initialEntries={['/']}>
                <I18nextProvider i18n={i18n}>
                    <AuthProvider>
                        <AppContent />
                    </AuthProvider>
                </I18nextProvider>
            </MemoryRouter>
        );
        expect(await screen.findByRole('heading', { name: 'One MCP' })).toBeInTheDocument();
        expect(await screen.findByRole('heading', { name: 'dashboard.title' })).toBeInTheDocument();
        expect(screen.queryByText('Loading authentication...')).not.toBeInTheDocument();
        if (signedIn) expect(screen.getByText('Fixture User')).toBeInTheDocument();
        else expect(screen.getByRole('button', { name: 'auth.loginButton' })).toBeInTheDocument();
    });
});
