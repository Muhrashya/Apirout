import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './auth/AuthProvider'
import { ToastProvider } from './components/Toast'
import { ErrorBoundary } from './components/ErrorBoundary'
import { ProtectedRoute, SettingsGuard } from './routing/guards'
import AppLayout from './layout/AppLayout'
import { BrandingProvider } from './context/BrandingContext'

// Lazy-loaded route pages for code-splitting and smaller initial bundle
const Login = lazy(() => import('./pages/Login'))
const Dashboard = lazy(() => import('./pages/Dashboard'))
const RoutesPage = lazy(() => import('./pages/Routes'))
const RouteFormPage = lazy(() => import('./pages/routes/RouteFormPage'))
const ServicesPage = lazy(() => import('./pages/Services'))
const ServiceFormPage = lazy(() => import('./pages/services/ServiceFormPage'))
const UpstreamsPage = lazy(() => import('./pages/Upstreams'))
const UpstreamFormPage = lazy(() => import('./pages/upstreams/UpstreamFormPage'))
const QueriesPage = lazy(() => import('./pages/Queries'))
const ConsumersPage = lazy(() => import('./pages/Consumers'))
const ConsumerFormPage = lazy(() => import('./pages/consumers/ConsumerFormPage'))
const InstantAPI = lazy(() => import('./pages/InstantAPI'))
const Traffic = lazy(() => import('./pages/observability/Traffic'))
const ApiLogs = lazy(() => import('./pages/observability/ApiLogs'))
const Analytics = lazy(() => import('./pages/observability/Analytics'))
const Profile = lazy(() => import('./pages/settings/Profile'))
const Connections = lazy(() => import('./pages/settings/Connections'))
const ConnectionFormPage = lazy(() => import('./pages/settings/ConnectionFormPage'))
const ApiClients = lazy(() => import('./pages/settings/ApiClients'))
const Users = lazy(() => import('./pages/settings/Users'))
const UserFormPage = lazy(() => import('./pages/settings/UserFormPage'))
const Domains = lazy(() => import('./pages/settings/Domains'))
const DomainFormPage = lazy(() => import('./pages/settings/DomainFormPage'))
const ActivityLogs = lazy(() => import('./pages/settings/ActivityLogs'))
const BrandingPage = lazy(() => import('./pages/settings/Branding'))
const ApiCollections = lazy(() => import('./pages/settings/ApiCollections'))

function PageFallback() {
    return (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '300px' }}>
            <div className="spinner"></div>
        </div>
    )
}

function App() {
    return (
        <ErrorBoundary>
            <BrowserRouter>
                <ToastProvider>
                    <AuthProvider>
                        <BrandingProvider>
                            <Suspense fallback={<PageFallback />}>
                                <Routes>
                                    <Route path="/login" element={<Login />} />

                                    <Route element={<ProtectedRoute />}>
                                        <Route element={<AppLayout />}>
                                            <Route path="/overview" element={<Dashboard />} />
                                            <Route path="/dashboard" element={<Navigate to="/overview" replace />} />

                                            {/* Gateway Core Entities (Kong Architecture) */}
                                            <Route path="/routes" element={<RoutesPage />} />
                                            <Route path="/routes/new" element={<RouteFormPage />} />
                                            <Route path="/routes/:id/edit" element={<RouteFormPage />} />
                                            <Route path="/endpoints" element={<Navigate to="/routes" replace />} />

                                            <Route path="/services" element={<ServicesPage />} />
                                            <Route path="/services/new" element={<ServiceFormPage />} />
                                            <Route path="/services/:id/edit" element={<ServiceFormPage />} />

                                            <Route path="/upstreams" element={<UpstreamsPage />} />
                                            <Route path="/upstreams/new" element={<UpstreamFormPage />} />
                                            <Route path="/upstreams/:id/edit" element={<UpstreamFormPage />} />
                                            <Route path="/queries" element={<QueriesPage />} />

                                            <Route path="/consumers" element={<ConsumersPage />} />
                                            <Route path="/consumers/new" element={<ConsumerFormPage />} />
                                            <Route path="/consumers/:id/edit" element={<ConsumerFormPage />} />

                                            <Route path="/instant-api" element={<InstantAPI />} />

                                            {/* Observability (Kong Style) */}
                                            <Route path="/traffic" element={<Traffic />} />
                                            <Route path="/logs" element={<ApiLogs />} />
                                            <Route path="/requests" element={<Navigate to="/logs" replace />} />
                                            <Route path="/analytics" element={<Analytics />} />
                                            <Route path="/docs" element={<Navigate to="/settings/collections" replace />} />
                                            <Route path="/audit-logs" element={<Navigate to="/settings/activity-logs" replace />} />

                                            {/* Profile Settings (Available to all authenticated users) */}
                                            <Route path="/settings/profile" element={<Profile />} />

                                            {/* Console Admin Settings */}
                                            <Route path="/settings" element={<SettingsRedirect />} />
                                            <Route path="/settings/console" element={<SettingsRedirect />} />
                                            <Route element={<SettingsGuard />}>
                                                <Route path="/settings/connections" element={<Connections />} />
                                                <Route path="/settings/connections/new" element={<ConnectionFormPage />} />
                                                <Route path="/settings/connections/:id/edit" element={<ConnectionFormPage />} />

                                                <Route path="/settings/domains" element={<Domains />} />
                                                <Route path="/settings/domains/new" element={<DomainFormPage />} />
                                                <Route path="/settings/domains/:id/edit" element={<DomainFormPage />} />

                                                <Route path="/settings/branding" element={<BrandingPage />} />
                                                <Route path="/settings/collections" element={<ApiCollections />} />
                                                <Route path="/settings/api-clients" element={<ApiClients />} />
                                                <Route path="/settings/groups" element={<Navigate to="/settings/users" replace />} />

                                                <Route path="/settings/users" element={<Users />} />
                                                <Route path="/settings/users/new" element={<UserFormPage />} />
                                                <Route path="/settings/users/:id/edit" element={<UserFormPage />} />

                                                <Route path="/settings/activity-logs" element={<ActivityLogs />} />
                                            </Route>
                                        </Route>
                                    </Route>

                                    <Route path="/" element={<RootRedirect />} />
                                    <Route path="*" element={<Navigate to="/" replace />} />
                                </Routes>
                            </Suspense>
                        </BrandingProvider>
                    </AuthProvider>
                </ToastProvider>
            </BrowserRouter>
        </ErrorBoundary>
    )
}

function SettingsRedirect() {
    const { user, canManageEndpoints } = useAuth()
    const canAccessSettings = user?.role === 'SUPER_ADMIN' || user?.role === 'ADMIN' || canManageEndpoints()
    return <Navigate to={canAccessSettings ? "/settings/connections" : "/settings/profile"} replace />
}

function RootRedirect() {
    const { isAuthenticated, isLoading } = useAuth()

    if (isLoading) {
        return (
            <div className="loading" style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <div className="spinner"></div>
            </div>
        )
    }

    return <Navigate to={isAuthenticated ? "/overview" : "/login"} replace />
}

export default App
