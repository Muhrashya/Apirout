import { useState, useEffect, useRef } from 'react';
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../auth/AuthProvider';
import { useBranding } from '../context/BrandingContext';
import {
    IconOverview, IconServices, IconRoutes, IconUpstreams, IconConsumers,
    IconTraffic, IconFileText, IconChartBar,
    IconMenu, IconBolt, IconArrowLeft, IconDatabase, IconCloud, IconUser,
    IconKey, IconAudit, IconPalette, IconCollection, IconSettings, IconLogout, IconQuery
} from '../components/Icons';
import './AppLayout.css';

export default function AppLayout() {
    const {
        user,
        logout,
        isSuperAdmin,
        isAdmin,
        isDeveloper,
        isMaintainer,
        canAccessUpstreams,
        canAccessQueries,
        canAccessInstantApi,
        canAccessDomains,
        canManageUsers,
    } = useAuth();
    const { branding } = useBranding();
    const navigate = useNavigate();
    const location = useLocation();

    // Determine if in Settings mode
    const isSettingsMode = location.pathname.startsWith('/settings');

    // Sidebar state
    const [isSidebarOpen, setIsSidebarOpen] = useState(window.innerWidth > 900);
    // User Menu dropdown state
    const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
    const userMenuRef = useRef<HTMLDivElement>(null);

    // Enforce Light Mode
    useEffect(() => {
        document.documentElement.setAttribute('data-theme', 'light');
        localStorage.setItem('theme', 'light');
    }, []);

    // Close dropdowns on outside click
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (userMenuRef.current && !userMenuRef.current.contains(event.target as Node)) {
                setIsUserMenuOpen(false);
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    // Update sidebar on window resize
    useEffect(() => {
        const handleResize = () => {
            if (window.innerWidth > 900) {
                setIsSidebarOpen(true);
            } else {
                setIsSidebarOpen(false);
            }
        };

        window.addEventListener('resize', handleResize);
        return () => window.removeEventListener('resize', handleResize);
    }, []);

    const toggleSidebar = () => {
        setIsSidebarOpen(prev => !prev);
    };

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    const roleUpper = (user?.role || '').toUpperCase();
    const roleClass = roleUpper === 'SUPER_ADMIN' ? 'super-admin'
        : roleUpper === 'ADMIN' ? 'admin'
        : roleUpper === 'DEVELOPER' ? 'developer'
        : roleUpper === 'MAINTAINER' ? 'maintainer'
        : 'developer';

    const roleLabel = roleUpper === 'SUPER_ADMIN' ? 'Super Admin'
        : roleUpper === 'ADMIN' ? 'Admin'
        : roleUpper === 'DEVELOPER' ? 'Developer'
        : roleUpper === 'MAINTAINER' ? 'Maintainer'
        : 'Developer';

    // Calculate avatar initials
    const initials = user?.email
        ? user.email.substring(0, 2).toUpperCase()
        : 'SA';

    return (
        <div className="app-layout">
            <header className="topbar">
                <div className="topbar-left">
                    <button className="menu-toggle" onClick={toggleSidebar} aria-label="Toggle Navigation">
                        <IconMenu width={20} height={20} />
                    </button>
                    <div className="topbar-brand" onClick={() => navigate('/overview')} style={{ cursor: 'pointer' }}>
                        <div className="brand-logo-box">
                            <img
                                src={branding.logo_url}
                                alt={branding.app_name}
                                style={{ width: '36px', height: '36px', objectFit: 'contain' }}
                                onError={(e) => { (e.target as HTMLImageElement).src = '/logo.png'; }}
                            />
                        </div>
                        <div className="brand-text-col">
                            <span className="brand-name">{branding.app_name}</span>
                            <span className="brand-tagline">{branding.tagline}</span>
                        </div>
                    </div>
                </div>

                <div className="topbar-right">
                    {/* User Avatar & Dropdown */}
                    <div className="user-dropdown-wrapper" ref={userMenuRef}>
                        <button
                            className="user-avatar-btn"
                            onClick={() => setIsUserMenuOpen(prev => !prev)}
                            title={user?.email}
                        >
                            {initials}
                        </button>
                        {isUserMenuOpen && (
                            <div className="user-dropdown-card">
                                <div className="user-dropdown-header">
                                    <div className="user-dropdown-avatar">
                                        {initials}
                                    </div>
                                    <div className="user-dropdown-info">
                                        <div className="user-dropdown-email" title={user?.email}>
                                            {user?.email}
                                        </div>
                                        <span className={`user-dropdown-badge ${roleClass}`}>
                                            {roleLabel}
                                        </span>
                                    </div>
                                </div>

                                <div className="user-dropdown-divider" />

                                <div className="user-dropdown-menu">
                                    <NavLink
                                        to="/settings"
                                        className="user-dropdown-item"
                                        onClick={() => setIsUserMenuOpen(false)}
                                    >
                                        <span className="user-dropdown-item-icon">
                                            <IconSettings width={16} height={16} />
                                        </span>
                                        <span>Settings</span>
                                    </NavLink>
                                </div>

                                <div className="user-dropdown-divider" />

                                <button className="user-dropdown-logout-btn" onClick={handleLogout}>
                                    <span className="user-dropdown-item-icon">
                                        <IconLogout width={16} height={16} />
                                    </span>
                                    <span>Log out</span>
                                </button>
                            </div>
                        )}
                    </div>
                </div>
            </header>

            <div className="app-body">
                <div
                    className={`sidebar-overlay ${isSidebarOpen ? 'open' : ''}`}
                    onClick={() => setIsSidebarOpen(false)}
                />

                <aside className={`sidebar ${isSidebarOpen ? 'open' : 'closed'}`}>
                    <div className="sidebar-scrollable">
                        {isSettingsMode ? (
                            /* Context: SETTINGS & ADMINISTRATION */
                            <>
                                <NavLink to="/overview" className="sidebar-back-btn">
                                    <IconArrowLeft width={15} height={15} />
                                    <span>Back to Gateway</span>
                                </NavLink>

                                {(isSuperAdmin || isAdmin || isDeveloper || isMaintainer) && (
                                    <div className="nav-group-section">
                                        <div className="nav-group-title">ADMIN CONSOLE</div>
                                        <ul className="nav-list">
                                            {(isSuperAdmin || isAdmin || isDeveloper) && (
                                                <li>
                                                    <NavLink to="/settings/connections" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
                                                        <span className="nav-icon"><IconDatabase width={18} height={18} /></span>
                                                        <span className="nav-label">Database Sources</span>
                                                    </NavLink>
                                                </li>
                                            )}
                                            {canAccessDomains() && (
                                                <li>
                                                    <NavLink to="/settings/domains" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
                                                        <span className="nav-icon"><IconCloud width={18} height={18} /></span>
                                                        <span className="nav-label">Domains &amp; TLS</span>
                                                    </NavLink>
                                                </li>
                                            )}
                                            {(isSuperAdmin || isAdmin) && (
                                                <li>
                                                    <NavLink to="/settings/branding" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
                                                        <span className="nav-icon"><IconPalette width={18} height={18} /></span>
                                                        <span className="nav-label">Customization</span>
                                                    </NavLink>
                                                </li>
                                            )}
                                            {(isSuperAdmin || isAdmin || isDeveloper) && (
                                                <li>
                                                    <NavLink to="/settings/collections" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
                                                        <span className="nav-icon"><IconCollection width={18} height={18} /></span>
                                                        <span className="nav-label">API Collections</span>
                                                    </NavLink>
                                                </li>
                                            )}
                                        </ul>
                                    </div>
                                )}

                                <div className="nav-group-section">
                                    <div className="nav-group-title">ACCESS &amp; SECURITY</div>
                                    <ul className="nav-list">
                                        {canManageUsers() && (
                                            <li>
                                                <NavLink to="/settings/users" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
                                                    <span className="nav-icon"><IconUser width={18} height={18} /></span>
                                                    <span className="nav-label">Users</span>
                                                </NavLink>
                                            </li>
                                        )}
                                        <li>
                                            <NavLink to="/settings/profile" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
                                                <span className="nav-icon"><IconKey width={18} height={18} /></span>
                                                <span className="nav-label">Profile &amp; Password</span>
                                            </NavLink>
                                        </li>
                                    </ul>
                                </div>

                                {(isSuperAdmin || isAdmin || isMaintainer) && (
                                    <div className="nav-group-section">
                                        <div className="nav-group-title">AUDIT TRAIL</div>
                                        <ul className="nav-list">
                                            <li>
                                                <NavLink to="/settings/activity-logs" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
                                                    <span className="nav-icon"><IconAudit width={18} height={18} /></span>
                                                    <span className="nav-label">Activity Logs</span>
                                                </NavLink>
                                            </li>
                                        </ul>
                                    </div>
                                )}
                            </>
                        ) : (
                            /* Context: GATEWAY & OBSERVABILITY */
                            <>
                                <div className="nav-group-section">
                                    <div className="nav-group-title">GATEWAY</div>
                                    <ul className="nav-list">
                                        <li>
                                            <NavLink
                                                to="/overview"
                                                className={({ isActive }) => isActive || location.pathname === '/dashboard' ? 'nav-link active' : 'nav-link'}
                                            >
                                                <span className="nav-icon"><IconOverview width={18} height={18} /></span>
                                                <span className="nav-label">Overview</span>
                                            </NavLink>
                                        </li>
                                        <li>
                                            <NavLink
                                                to="/routes"
                                                className={({ isActive }) => isActive || location.pathname.startsWith('/routes') || location.pathname === '/endpoints' ? 'nav-link active' : 'nav-link'}
                                            >
                                                <span className="nav-icon"><IconRoutes width={18} height={18} /></span>
                                                <span className="nav-label">Routes</span>
                                            </NavLink>
                                        </li>
                                        {(isSuperAdmin || isAdmin || isMaintainer) && (
                                            <li>
                                                <NavLink
                                                    to="/services"
                                                    className={({ isActive }) => isActive || location.pathname.startsWith('/services') ? 'nav-link active' : 'nav-link'}
                                                >
                                                    <span className="nav-icon"><IconServices width={18} height={18} /></span>
                                                    <span className="nav-label">Services</span>
                                                </NavLink>
                                            </li>
                                        )}
                                        {canAccessUpstreams() && (
                                            <li>
                                                <NavLink
                                                    to="/upstreams"
                                                    className={({ isActive }) => isActive || location.pathname.startsWith('/upstreams') ? 'nav-link active' : 'nav-link'}
                                                >
                                                    <span className="nav-icon"><IconUpstreams width={18} height={18} /></span>
                                                    <span className="nav-label">Upstreams</span>
                                                </NavLink>
                                            </li>
                                        )}
                                        <li>
                                            <NavLink
                                                to="/consumers"
                                                className={({ isActive }) => isActive || location.pathname.startsWith('/consumers') ? 'nav-link active' : 'nav-link'}
                                            >
                                                <span className="nav-icon"><IconConsumers width={18} height={18} /></span>
                                                <span className="nav-label">Consumers</span>
                                            </NavLink>
                                        </li>
                                        {canAccessInstantApi() && (
                                            <li>
                                                <NavLink
                                                    to="/instant-api"
                                                    className={({ isActive }) => isActive || location.pathname.startsWith('/instant-api') ? 'nav-link active' : 'nav-link'}
                                                >
                                                    <span className="nav-icon"><IconBolt width={18} height={18} /></span>
                                                    <span className="nav-label">Instant API</span>
                                                </NavLink>
                                            </li>
                                        )}
                                        {canAccessQueries() && (
                                            <li>
                                                <NavLink
                                                    to="/queries"
                                                    className={({ isActive }) => isActive || location.pathname.startsWith('/queries') ? 'nav-link active' : 'nav-link'}
                                                >
                                                    <span className="nav-icon"><IconQuery width={18} height={18} /></span>
                                                    <span className="nav-label">Queries</span>
                                                </NavLink>
                                            </li>
                                        )}
                                    </ul>
                                </div>

                                <div className="nav-group-section">
                                    <div className="nav-group-title">OBSERVABILITY</div>
                                    <ul className="nav-list">
                                        {(isSuperAdmin || isAdmin || isMaintainer) && (
                                            <li>
                                                <NavLink to="/traffic" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
                                                    <span className="nav-icon"><IconTraffic width={18} height={18} /></span>
                                                    <span className="nav-label">Traffic</span>
                                                </NavLink>
                                            </li>
                                        )}
                                        <li>
                                            <NavLink to="/logs" className={({ isActive }) => isActive || location.pathname === '/requests' ? 'nav-link active' : 'nav-link'}>
                                                <span className="nav-icon"><IconFileText width={18} height={18} /></span>
                                                <span className="nav-label">API Logs</span>
                                            </NavLink>
                                        </li>
                                        {(isSuperAdmin || isAdmin || isMaintainer) && (
                                            <li>
                                                <NavLink to="/analytics" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
                                                    <span className="nav-icon"><IconChartBar width={18} height={18} /></span>
                                                    <span className="nav-label">Analytics</span>
                                                </NavLink>
                                            </li>
                                        )}
                                    </ul>
                                </div>
                            </>
                        )}
                    </div>
                </aside>

                <main className="main-content">
                    <Outlet />
                </main>
            </div>
        </div>
    );
}
