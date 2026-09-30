import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { apiClient, setToken, clearToken, hasToken } from '../api/apiClient';

export interface GroupMembership {
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

export type UserRole = 'SUPER_ADMIN' | 'ADMIN' | 'DEVELOPER' | 'MAINTAINER' | string;

export interface User {
    id: string;
    email: string;
    role: UserRole;
    groups?: GroupMembership[];
}

export interface AuthContextType {
    user: User | null;
    isLoading: boolean;
    isAuthenticated: boolean;
    isSuperAdmin: boolean;
    isAdmin: boolean;
    isDeveloper: boolean;
    isMaintainer: boolean;
    login: (email: string, password: string) => Promise<void>;
    logout: () => void;
    refreshUser: () => Promise<void>;
    canManageEndpoints: () => boolean;
    canManageConnections: () => boolean;
    canAccessUpstreams: () => boolean;
    canAccessQueries: () => boolean;
    canAccessInstantApi: () => boolean;
    canAccessDomains: () => boolean;
    canManageUsers: () => boolean;
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

    // Role booleans
    const roleUpper = (user?.role || '').toUpperCase();
    const isSuperAdmin = roleUpper === 'SUPER_ADMIN';
    const isAdmin = roleUpper === 'ADMIN';
    const isDeveloper = roleUpper === 'DEVELOPER';
    const isMaintainer = roleUpper === 'MAINTAINER';

    // Users Management is root only
    const canManageUsers = (): boolean => isSuperAdmin;

    // Upstreams & Targets control: SRE maintainers, Admins, Super Admins
    const canAccessUpstreams = (): boolean => isSuperAdmin || isAdmin || isMaintainer;

    // Domains & TLS certificates: Maintainers, Admins, Super Admins
    const canAccessDomains = (): boolean => isSuperAdmin || isAdmin || isMaintainer;

    // Queries & SQL preview: Developers, Admins, Super Admins
    const canAccessQueries = (): boolean => isSuperAdmin || isAdmin || isDeveloper;

    // Instant REST API generator: Developers, Admins, Super Admins
    const canAccessInstantApi = (): boolean => isSuperAdmin || isAdmin || isDeveloper;

    // Check if user can manage endpoints (publish/edit routes)
    const canManageEndpoints = (): boolean => {
        if (!user) return false;
        if (isSuperAdmin || isAdmin || isDeveloper) return true;
        return user.groups?.some(g => g.can_manage_endpoints) || false;
    };

    // Check if user can manage connections (database sources)
    const canManageConnections = (): boolean => {
        if (!user) return false;
        if (isSuperAdmin || isAdmin) return true;
        return user.groups?.some(g => g.can_manage_connections) || false;
    };

    // Get groups where user can manage endpoints
    const getManageableGroups = (): GroupMembership[] => {
        if (!user) return [];
        if (isSuperAdmin) {
            return user.groups || [];
        }
        return user.groups?.filter(g => g.can_manage_endpoints) || [];
    };

    return (
        <AuthContext.Provider
            value={{
                user,
                isLoading,
                isAuthenticated: !!user,
                isSuperAdmin,
                isAdmin,
                isDeveloper,
                isMaintainer,
                login,
                logout,
                refreshUser,
                canManageEndpoints,
                canManageConnections,
                canAccessUpstreams,
                canAccessQueries,
                canAccessInstantApi,
                canAccessDomains,
                canManageUsers,
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
