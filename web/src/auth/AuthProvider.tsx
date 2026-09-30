import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { apiClient, setToken, clearToken, hasToken } from '../api/apiClient';

interface GroupMembership {
    group_id: string;
    group_name: string;
    can_manage_connections: boolean;
    can_manage_queries: boolean;
    can_manage_endpoints: boolean;
    can_execute_endpoints: boolean;
    can_view_docs: boolean;
    can_view_dashboard: boolean;
    can_view_audit_logs: boolean;
}

interface User {
    id: string;
    email: string;
    role: string;
    groups?: GroupMembership[];
}

interface AuthContextType {
    user: User | null;
    isLoading: boolean;
    isAuthenticated: boolean;
    login: (email: string, password: string) => Promise<void>;
    logout: () => void;
    refreshUser: () => Promise<void>;
    canManageEndpoints: () => boolean;
    canManageConnections: () => boolean;
    getManageableGroups: () => GroupMembership[];
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null);
    const [isLoading, setIsLoading] = useState(true);

    const refreshUser = async () => {
        try {
            const response = await apiClient.get<User>('/auth/me');
            setUser(response);
        } catch {
            setUser(null);
            clearToken();
        }
    };

    useEffect(() => {
        const init = async () => {
            if (hasToken()) {
                await refreshUser();
            }
            setIsLoading(false);
        };
        init();
    }, []);

    const login = async (email: string, password: string) => {
        const response = await apiClient.post<{ token: string; user: User }>('/auth/login', {
            email,
            password,
        });

        // Store token first
        setToken(response.token);

        // Then update user state
        setUser(response.user);

        // Fetch fresh user data to ensure state is synced
        await refreshUser();
    };

    const logout = () => {
        clearToken();
        setUser(null);
    };

    // Check if user can manage endpoints (publish)
    const canManageEndpoints = (): boolean => {
        if (!user) return false;
        if (user.role === 'SUPER_ADMIN') return true;
        // Check if any group has can_manage_endpoints permission
        return user.groups?.some(g => g.can_manage_endpoints) || false;
    };

    // Check if user can manage connections
    const canManageConnections = (): boolean => {
        if (!user) return false;
        if (user.role === 'SUPER_ADMIN' || user.role === 'ADMIN') return true;
        return user.groups?.some(g => g.can_manage_connections) || false;
    };

    // Get groups where user can manage endpoints
    const getManageableGroups = (): GroupMembership[] => {
        if (!user) return [];
        if (user.role === 'SUPER_ADMIN') {
            // SUPER_ADMIN can manage all groups - return all groups user has access to
            return user.groups || [];
        }
        // Return only groups where can_manage_endpoints is true
        return user.groups?.filter(g => g.can_manage_endpoints) || [];
    };

    return (
        <AuthContext.Provider
            value={{
                user,
                isLoading,
                isAuthenticated: !!user,
                login,
                logout,
                refreshUser,
                canManageEndpoints,
                canManageConnections,
                getManageableGroups,
            }}
        >
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth() {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
}
