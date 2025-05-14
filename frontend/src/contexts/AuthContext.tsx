import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

// Simplified User type based on what's stored and needed for UI
// Ensure this matches or is a subset of backend's model.User structure returned in login response
interface User {
    id: number | string; // Or just string if backend ID is always string
    username: string;
    displayName?: string;
    email?: string;
    role?: number;
    // Add other fields if needed by the UI, e.g., avatar_url
}

interface AuthContextType {
    currentUser: User | null;
    token: string | null;
    login: (userData: User, authToken: string) => void;
    logout: () => void;
    isLoading: boolean; // To handle initial loading of auth state
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
    const [currentUser, setCurrentUser] = useState<User | null>(null);
    const [token, setToken] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        // Try to load user and token from localStorage on initial render
        try {
            const storedToken = localStorage.getItem('token');
            const storedUserString = localStorage.getItem('user');
            if (storedToken && storedUserString) {
                const storedUser = JSON.parse(storedUserString) as User;
                setToken(storedToken);
                setCurrentUser(storedUser);
            }
        } catch (error) {
            console.error("Error parsing user from localStorage", error);
            // Clear potentially corrupted storage
            localStorage.removeItem('token');
            localStorage.removeItem('user');
        }
        setIsLoading(false);
    }, []);

    const login = (userData: User, authToken: string) => {
        localStorage.setItem('token', authToken);
        localStorage.setItem('user', JSON.stringify(userData));
        setToken(authToken);
        setCurrentUser(userData);
    };

    const logout = () => {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        setToken(null);
        setCurrentUser(null);
        // Optionally, navigate to login page or home page after logout
        // navigate('/login'); // Requires useNavigate from react-router-dom
    };

    return (
        <AuthContext.Provider value={{ currentUser, token, login, logout, isLoading }}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (context === undefined) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
}; 