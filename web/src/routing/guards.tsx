import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../auth/AuthProvider';
import { useToast } from '../components/Toast';
import { useEffect } from 'react';

export function ProtectedRoute() {
    const { isAuthenticated, isLoading, setupRequired } = useAuth();

    if (isLoading) {
        return (
            <div className="loading" style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <div className="spinner"></div>
            </div>
        );
    }

    if (setupRequired) {
        return <Navigate to="/setup" replace />;
    }

    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }

    return <Outlet />;
}

export function SetupGuard() {
    const { setupRequired, isLoading } = useAuth();

    if (isLoading) {
        return (
            <div className="loading" style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <div className="spinner"></div>
            </div>
        );
    }

    // If setup is already complete, permanently lock out /setup and redirect to /login
    if (!setupRequired) {
        return <Navigate to="/login" replace />;
    }

    return <Outlet />;
}

export function PublicRoute() {
    const { setupRequired, isAuthenticated, isLoading } = useAuth();

    if (isLoading) {
        return (
            <div className="loading" style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <div className="spinner"></div>
            </div>
        );
    }

    if (setupRequired) {
        return <Navigate to="/setup" replace />;
    }

    if (isAuthenticated) {
        return <Navigate to="/overview" replace />;
    }

    return <Outlet />;
}

export function SettingsGuard() {
    const { user, canManageEndpoints } = useAuth();
    const toast = useToast();

    const roleUpper = (user?.role || '').toUpperCase();
    const hasAccess = roleUpper === 'SUPER_ADMIN' || roleUpper === 'ADMIN' || roleUpper === 'MAINTAINER' || roleUpper === 'DEVELOPER' || canManageEndpoints();

    useEffect(() => {
        if (!hasAccess && user) {
            toast.error('No permission to access Settings');
        }
    }, [hasAccess, user, toast]);

    if (!hasAccess) {
        return <Navigate to="/overview" replace />;
    }

    return <Outlet />;
}

export function RoleRoute({ allowedRoles, fallback = "/overview" }: { allowedRoles: string[]; fallback?: string }) {
    const { user, isLoading } = useAuth();
    const toast = useToast();

    if (isLoading) {
        return (
            <div className="loading" style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <div className="spinner"></div>
            </div>
        );
    }

    const roleUpper = (user?.role || '').toUpperCase();
    const hasAccess = !!user && allowedRoles.includes(roleUpper);

    useEffect(() => {
        if (!hasAccess && user) {
            toast.error(`Access restricted for role "${user.role}"`);
        }
    }, [hasAccess, user, toast]);

    if (!hasAccess) {
        return <Navigate to={fallback} replace />;
    }

    return <Outlet />;
}
