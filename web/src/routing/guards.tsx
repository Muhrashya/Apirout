import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../auth/AuthProvider';
import { useToast } from '../components/Toast';
import { useEffect } from 'react';

export function ProtectedRoute() {
    const { isAuthenticated, isLoading } = useAuth();

    if (isLoading) {
        return (
            <div className="loading" style={{ height: '100vh' }}>
                <div className="spinner"></div>
            </div>
        );
    }

    if (!isAuthenticated) {
        return <Navigate to="/login" replace />;
    }

    return <Outlet />;
}

export function SettingsGuard() {
    const { user, canManageEndpoints } = useAuth();
    const toast = useToast();

    const hasAccess = user?.role === 'SUPER_ADMIN' || user?.role === 'ADMIN' || canManageEndpoints();

    useEffect(() => {
        if (!hasAccess) {
            toast.error('No permission to access Settings');
        }
    }, [hasAccess, toast]);

    if (!hasAccess) {
        return <Navigate to="/dashboard" replace />;
    }

    return <Outlet />;
}
