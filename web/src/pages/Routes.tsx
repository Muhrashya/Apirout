import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { gatewayApi, getGatewayBaseUrl, Route, GatewayService } from '../api/gatewayApi';
import { apiClient } from '../api/apiClient';
import { useToast } from '../components/Toast';
import { Modal } from '../components/Modal';
import { useAuth } from '../auth/AuthProvider';
import {
    IconRoutes,
    IconSearch,
    IconRefresh,
    IconPlus,
    IconPencil,
    IconTrash,
    IconEye,
    IconServer,
    IconLayers,
    IconDatabase,
    IconCheck,
    IconCopy
} from '../components/Icons';

export default function RoutesPage() {
    const [routes, setRoutes] = useState<Route[]>([]);
    const [services, setServices] = useState<GatewayService[]>([]);
    const [globalRateLimit, setGlobalRateLimit] = useState<number | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const [selectedService, setSelectedService] = useState<string>('all');
    const [selectedMethod, setSelectedMethod] = useState<string>('all');

    const [viewDetails, setViewDetails] = useState<Route | null>(null);
    const [copiedCurl, setCopiedCurl] = useState(false);

    const inputRef = useRef<HTMLInputElement>(null);
    const { user, canManageEndpoints } = useAuth();
    const toast = useToast();

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                inputRef.current?.focus();
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, []);

    const loadData = async () => {
        setIsLoading(true);
        try {
            const [routesRes, servicesRes, settingsRes] = await Promise.all([
                gatewayApi.getRoutes().catch(() => ({ data: [] })),
                gatewayApi.getServices().catch(() => ({ data: [] })),
                apiClient.get<any>('/settings').catch(() => null),
            ]);

            const routesList: Route[] = Array.isArray(routesRes)
                ? routesRes
                : (Array.isArray(routesRes?.data) ? routesRes.data : []);
            const servicesList: GatewayService[] = Array.isArray(servicesRes)
                ? servicesRes
                : (Array.isArray(servicesRes?.data) ? servicesRes.data : []);

            setRoutes(routesList);
            setServices(servicesList);

            if (settingsRes?.global_rate_limit_rpm) {
                setGlobalRateLimit(settingsRes.global_rate_limit_rpm);
            }
        } catch (err: any) {
            toast.error(err?.message || 'Failed to load routes');
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        loadData();
    }, []);

    const navigate = useNavigate();

    const openCreateModal = () => {
        navigate('/routes/new');
    };

    const openEditModal = (route: Route) => {
        navigate(`/routes/${route.id}/edit`);
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure you want to delete this route? Incoming traffic matching this path will receive 404.')) return;
        try {
            await gatewayApi.deleteRoute(id);
            toast.success('Route deleted successfully');
            loadData();
        } catch (err: any) {
            toast.error(err?.message || 'Failed to delete route');
        }
    };

    const toggleEnabled = async (route: Route) => {
        try {
            await gatewayApi.updateRoute(route.id, {
                enabled: !route.enabled,
            });
            toast.success(`Route ${!route.enabled ? 'enabled' : 'disabled'}`);
            loadData();
        } catch (err: any) {
            toast.error(err?.message || 'Failed to toggle route status');
        }
    };

    const canManage = user?.role === 'SUPER_ADMIN' || user?.role === 'ADMIN' || canManageEndpoints();

    const filteredRoutes = routes.filter(r => {
        const matchesSearch =
            r.path.toLowerCase().includes(searchTerm.toLowerCase()) ||
            (r.name && r.name.toLowerCase().includes(searchTerm.toLowerCase())) ||
            r.method.toLowerCase().includes(searchTerm.toLowerCase()) ||
            (r.service_name && r.service_name.toLowerCase().includes(searchTerm.toLowerCase())) ||
            (r.description && r.description.toLowerCase().includes(searchTerm.toLowerCase()));

        const matchesService = selectedService === 'all' || r.service_id === selectedService;
        const matchesMethod = selectedMethod === 'all' || r.method === selectedMethod;

        return matchesSearch && matchesService && matchesMethod;
    });

    const getMethodBadgeStyle = (method: string) => {
        switch (method.toUpperCase()) {
            case 'GET':
                return { bg: '#dcfce7', color: '#16a34a' };
            case 'POST':
                return { bg: '#dbeafe', color: '#2563eb' };
            case 'PUT':
                return { bg: '#fef3c7', color: '#d97706' };
            case 'DELETE':
                return { bg: '#fee2e2', color: '#dc2626' };
            case 'PATCH':
                return { bg: '#f3e8ff', color: '#7e22ce' };
            default:
                return { bg: '#f1f5f9', color: '#475569' };
        }
    };

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        setCopiedCurl(true);
        setTimeout(() => setCopiedCurl(false), 2000);
    };

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Gateway</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Routes</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Gateway Routes</h1>
                    <p className="overview-subtitle">
                        High-performance request routing layer. Maps incoming HTTP requests to backend Services, Upstream pools, and Instant Database queries.
                    </p>
                </div>

                <div className="overview-header-actions">
                    <button className="btn-header-refresh" onClick={loadData} title="Reload routes">
                        <IconRefresh width={14} height={14} />
                        <span>Refresh</span>
                    </button>
                    {canManage && (
                        <button className="btn-header-new-route" onClick={openCreateModal}>
                            <IconPlus width={15} height={15} />
                            <span>New Route</span>
                        </button>
                    )}
                </div>
            </div>

            {/* Filter and Search Bar */}
            <div style={{
                backgroundColor: '#ffffff',
                border: '1px solid #e2e8f0',
                borderRadius: '12px',
                padding: '16px 20px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                flexWrap: 'wrap',
                gap: '12px',
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)',
                marginBottom: '16px'
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flex: '1 1 400px', flexWrap: 'wrap' }}>
                    <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        backgroundColor: '#f8fafc',
                        border: '1px solid #e2e8f0',
                        borderRadius: '8px',
                        padding: '0 12px',
                        width: '100%',
                        maxWidth: '360px',
                        height: '38px',
                    }}>
                        <IconSearch width={15} height={15} style={{ color: '#94a3b8', marginRight: '8px', flexShrink: 0 }} />
                        <input
                            ref={inputRef}
                            type="text"
                            placeholder="Search by path, name, service... (Ctrl+K)"
                            value={searchTerm}
                            onChange={e => setSearchTerm(e.target.value)}
                            style={{
                                border: 'none',
                                background: 'transparent',
                                fontSize: '13px',
                                outline: 'none',
                                width: '100%',
                                color: '#0f172a'
                            }}
                        />
                    </div>

                    <select
                        className="select select-filter"
                        value={selectedMethod}
                        onChange={e => setSelectedMethod(e.target.value)}
                    >
                        <option value="all">All Methods</option>
                        <option value="GET">GET</option>
                        <option value="POST">POST</option>
                        <option value="PUT">PUT</option>
                        <option value="DELETE">DELETE</option>
                        <option value="PATCH">PATCH</option>
                    </select>

                    <select
                        className="select select-filter"
                        style={{ maxWidth: '200px' }}
                        value={selectedService}
                        onChange={e => setSelectedService(e.target.value)}
                    >
                        <option value="all">All Services</option>
                        {services.map(s => (
                            <option key={s.id} value={s.id}>{s.name}</option>
                        ))}
                    </select>
                </div>

                <div style={{ fontSize: '13px', color: '#64748b' }}>
                    Showing <strong>{filteredRoutes.length}</strong> of {routes.length} routes
                </div>
            </div>

            {/* Routes Card / Table */}
            <div style={{
                backgroundColor: '#ffffff',
                border: '1px solid #e2e8f0',
                borderRadius: '12px',
                overflow: 'hidden',
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
            }}>
                {isLoading ? (
                    <div style={{ padding: '60px 24px', textAlign: 'center' }}>
                        <div className="spinner" style={{ margin: '0 auto 12px auto' }}></div>
                        <p style={{ color: '#64748b', fontSize: '14px' }}>Loading routes...</p>
                    </div>
                ) : filteredRoutes.length === 0 ? (
                    <div style={{ padding: '60px 24px', textAlign: 'center' }}>
                        <div style={{
                            width: '48px',
                            height: '48px',
                            borderRadius: '50%',
                            backgroundColor: '#f1f5f9',
                            color: '#94a3b8',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            margin: '0 auto 16px auto',
                        }}>
                            <IconRoutes width={24} height={24} />
                        </div>
                        <h3 style={{ fontSize: '16px', fontWeight: 600, color: '#0f172a', marginBottom: '6px' }}>
                            {routes.length === 0 ? 'No Routes Configured' : 'No Matching Routes'}
                        </h3>
                        <p style={{ fontSize: '13.5px', color: '#64748b', maxWidth: '420px', margin: '0 auto 20px auto' }}>
                            {routes.length === 0
                                ? 'Create your first Gateway route to route HTTP traffic to backend services or queries.'
                                : 'Try clearing your search filters to find the route you are looking for.'}
                        </p>
                        {canManage && routes.length === 0 && (
                            <button className="btn-header-new-route" onClick={openCreateModal}>
                                Create Route
                            </button>
                        )}
                    </div>
                ) : (
                    <div style={{ overflowX: 'auto' }}>
                        <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '13.5px' }}>
                            <thead>
                                <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>HTTP Method</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Route Path / Name</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Target Service</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Forwarding</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Rate Limit</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Access</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Status</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569', textAlign: 'right' }}>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredRoutes.map((route, idx) => {
                                    const mStyle = getMethodBadgeStyle(route.method);
                                    const matchedService = services.find(s => s.id === route.service_id);

                                    return (
                                        <tr
                                            key={route.id}
                                            style={{
                                                borderTop: idx > 0 ? '1px solid #f1f5f9' : 'none',
                                                backgroundColor: idx % 2 === 0 ? '#ffffff' : '#fafafa',
                                                transition: 'background 100ms ease'
                                            }}
                                        >
                                            {/* HTTP Method */}
                                            <td style={{ padding: '12px 20px' }}>
                                                <span style={{
                                                    fontSize: '11.5px',
                                                    fontWeight: 700,
                                                    padding: '3px 8px',
                                                    borderRadius: '6px',
                                                    backgroundColor: mStyle.bg,
                                                    color: mStyle.color,
                                                    fontFamily: 'monospace'
                                                }}>
                                                    {route.method}
                                                </span>
                                            </td>

                                            {/* Match Path & Name */}
                                            <td style={{ padding: '12px 20px' }}>
                                                <div style={{ display: 'flex', flexDirection: 'column' }}>
                                                    <code style={{
                                                        fontFamily: 'monospace',
                                                        fontSize: '13.5px',
                                                        fontWeight: 600,
                                                        color: '#0f172a'
                                                    }}>
                                                        {route.path}
                                                    </code>
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginTop: '2px' }}>
                                                        {route.name && route.name !== route.path && (
                                                            <span style={{ fontSize: '12px', fontWeight: 500, color: '#334155' }}>
                                                                {route.name}
                                                            </span>
                                                        )}
                                                        {route.description && (
                                                            <span style={{ fontSize: '11.5px', color: '#64748b' }}>
                                                                · {route.description}
                                                            </span>
                                                        )}
                                                    </div>
                                                </div>
                                            </td>

                                            {/* Target Service */}
                                            <td style={{ padding: '12px 20px', color: '#475569' }}>
                                                {matchedService ? (
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                        {matchedService.protocol === 'database' ? (
                                                            <IconDatabase width={14} height={14} style={{ color: '#0284c7' }} />
                                                        ) : matchedService.upstream_id ? (
                                                            <IconLayers width={14} height={14} style={{ color: '#7c3aed' }} />
                                                        ) : (
                                                            <IconServer width={14} height={14} style={{ color: '#2563eb' }} />
                                                        )}
                                                        <span style={{ fontWeight: 600, color: '#0f172a', fontSize: '13px' }}>
                                                            {matchedService.name}
                                                        </span>
                                                        <span style={{
                                                            fontSize: '10.5px',
                                                            padding: '1px 6px',
                                                            borderRadius: '4px',
                                                            backgroundColor: '#f1f5f9',
                                                            color: '#475569',
                                                            fontWeight: 600,
                                                            textTransform: 'uppercase'
                                                        }}>
                                                            {matchedService.upstream_id ? 'Upstream' : matchedService.protocol}
                                                        </span>
                                                    </div>
                                                ) : route.service_name ? (
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                        <IconServer width={14} height={14} style={{ color: '#64748b' }} />
                                                        <span style={{ fontWeight: 600, color: '#0f172a', fontSize: '13px' }}>
                                                            {route.service_name}
                                                        </span>
                                                    </div>
                                                ) : route.query_id ? (
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                        <IconDatabase width={14} height={14} style={{ color: '#0284c7' }} />
                                                        <span style={{ fontSize: '12.5px', color: '#0284c7', fontWeight: 500 }}>
                                                            SQL Instant API
                                                        </span>
                                                    </div>
                                                ) : (
                                                    <span style={{ fontSize: '12px', color: '#94a3b8', fontStyle: 'italic' }}>
                                                        Unassigned
                                                    </span>
                                                )}
                                            </td>

                                            {/* Forwarding Config */}
                                            <td style={{ padding: '12px 20px', color: '#475569', fontSize: '12px' }}>
                                                <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
                                                    {route.strip_path && (
                                                        <span style={{
                                                            padding: '2px 6px',
                                                            borderRadius: '4px',
                                                            backgroundColor: '#f1f5f9',
                                                            color: '#475569',
                                                            fontSize: '11px',
                                                            fontWeight: 600
                                                        }}>
                                                            strip_path
                                                        </span>
                                                    )}
                                                    {route.preserve_host && (
                                                        <span style={{
                                                            padding: '2px 6px',
                                                            borderRadius: '4px',
                                                            backgroundColor: '#eff6ff',
                                                            color: '#2563eb',
                                                            fontSize: '11px',
                                                            fontWeight: 600
                                                        }}>
                                                            preserve_host
                                                        </span>
                                                    )}
                                                    {!route.strip_path && !route.preserve_host && (
                                                        <span style={{ color: '#94a3b8', fontSize: '12px' }}>Default</span>
                                                    )}
                                                </div>
                                            </td>

                                            {/* Rate Limit */}
                                            <td style={{ padding: '12px 20px', color: '#475569', fontSize: '13px' }}>
                                                {route.rate_limit_enabled ? (
                                                    <span><strong>{route.rate_limit_rpm}</strong> rpm</span>
                                                ) : (
                                                    <span style={{ color: '#94a3b8' }}>Inherit ({globalRateLimit || 60} rpm)</span>
                                                )}
                                            </td>

                                            {/* Access (Public / Protected) */}
                                            <td style={{ padding: '12px 20px' }}>
                                                <span style={{
                                                    fontSize: '11px',
                                                    fontWeight: 600,
                                                    padding: '2px 8px',
                                                    borderRadius: '12px',
                                                    backgroundColor: route.is_public ? '#eff6ff' : '#f1f5f9',
                                                    color: route.is_public ? '#2563eb' : '#475569'
                                                }}>
                                                    {route.is_public ? 'Public' : 'Protected'}
                                                </span>
                                            </td>

                                            {/* Status Toggle */}
                                            <td style={{ padding: '12px 20px' }}>
                                                <button
                                                    onClick={() => canManage && toggleEnabled(route)}
                                                    style={{
                                                        border: 'none',
                                                        background: 'transparent',
                                                        cursor: canManage ? 'pointer' : 'default',
                                                        padding: 0
                                                    }}
                                                    title={canManage ? 'Click to toggle active state' : ''}
                                                >
                                                    <span style={{
                                                        fontSize: '11px',
                                                        fontWeight: 600,
                                                        padding: '2px 8px',
                                                        borderRadius: '12px',
                                                        backgroundColor: route.enabled ? '#dcfce7' : '#fee2e2',
                                                        color: route.enabled ? '#16a34a' : '#dc2626'
                                                    }}>
                                                        {route.enabled ? 'Active' : 'Disabled'}
                                                    </span>
                                                </button>
                                            </td>

                                            {/* Actions */}
                                            <td style={{ padding: '12px 20px', textAlign: 'right' }}>
                                                <div style={{ display: 'inline-flex', gap: '8px' }}>
                                                    <button
                                                        onClick={() => setViewDetails(route)}
                                                        title="View Route Details"
                                                        style={{
                                                            border: '1px solid #e2e8f0',
                                                            backgroundColor: '#ffffff',
                                                            padding: '6px',
                                                            borderRadius: '6px',
                                                            color: '#475569',
                                                            cursor: 'pointer'
                                                        }}
                                                    >
                                                        <IconEye width={14} height={14} />
                                                    </button>
                                                    {canManage && (
                                                        <button
                                                            onClick={() => openEditModal(route)}
                                                            title="Edit Route"
                                                            style={{
                                                                border: '1px solid #e2e8f0',
                                                                backgroundColor: '#ffffff',
                                                                padding: '6px',
                                                                borderRadius: '6px',
                                                                color: '#2563eb',
                                                                cursor: 'pointer'
                                                            }}
                                                        >
                                                            <IconPencil width={14} height={14} />
                                                        </button>
                                                    )}
                                                    {canManage && (
                                                        <button
                                                            onClick={() => handleDelete(route.id)}
                                                            title="Delete Route"
                                                            style={{
                                                                border: '1px solid #fee2e2',
                                                                backgroundColor: '#ffffff',
                                                                padding: '6px',
                                                                borderRadius: '6px',
                                                                color: '#dc2626',
                                                                cursor: 'pointer'
                                                            }}
                                                        >
                                                            <IconTrash width={14} height={14} />
                                                        </button>
                                                    )}
                                                </div>
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* Modal: View Route Details */}
            {viewDetails && (
                <Modal
                    isOpen={!!viewDetails}
                    onClose={() => setViewDetails(null)}
                    title={`Route: ${viewDetails.method} ${viewDetails.path}`}
                >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '14px', fontSize: '13px' }}>
                        <div>
                            <span style={{ color: '#64748b', display: 'block', fontSize: '12px' }}>Full Gateway URL</span>
                            <code style={{ backgroundColor: '#f1f5f9', padding: '6px 10px', borderRadius: '6px', display: 'block', marginTop: '4px', fontSize: '13px', fontWeight: 600 }}>
                                {getGatewayBaseUrl()}{viewDetails.path}
                            </code>
                        </div>

                        {/* Conceptual Chain: Route -> Service -> Upstream/Host */}
                        <div style={{
                            padding: '12px',
                            backgroundColor: '#eff6ff',
                            border: '1px solid #bfdbfe',
                            borderRadius: '8px'
                        }}>
                            <span style={{ fontSize: '12px', fontWeight: 600, color: '#1e40af', display: 'block', marginBottom: '4px' }}>
                                Gateway Execution Chain
                            </span>
                            <div style={{ fontSize: '12.5px', color: '#1e3a8a', display: 'flex', alignItems: 'center', gap: '6px' }}>
                                <span style={{ fontWeight: 600 }}>Route ({viewDetails.method} {viewDetails.path})</span>
                                <span>&rarr;</span>
                                <span style={{ fontWeight: 600 }}>Service ({viewDetails.service_name || 'Direct / Instant API'})</span>
                            </div>
                        </div>

                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
                            <div>
                                <span style={{ color: '#64748b', fontSize: '12px' }}>Access Mode</span>
                                <div style={{ fontWeight: 600, marginTop: '2px' }}>
                                    {viewDetails.is_public ? 'Public (Unrestricted)' : 'Protected (Requires Consumer API Key)'}
                                </div>
                            </div>
                            <div>
                                <span style={{ color: '#64748b', fontSize: '12px' }}>Rate Limit</span>
                                <div style={{ fontWeight: 600, marginTop: '2px' }}>
                                    {viewDetails.rate_limit_enabled ? `${viewDetails.rate_limit_rpm} RPM (Burst: ${viewDetails.rate_limit_burst})` : 'Inherited'}
                                </div>
                            </div>
                            <div>
                                <span style={{ color: '#64748b', fontSize: '12px' }}>Forwarding Settings</span>
                                <div style={{ fontWeight: 600, marginTop: '2px' }}>
                                    Strip Path: {viewDetails.strip_path ? 'Yes' : 'No'} | Preserve Host: {viewDetails.preserve_host ? 'Yes' : 'No'}
                                </div>
                            </div>
                            <div>
                                <span style={{ color: '#64748b', fontSize: '12px' }}>Protocols</span>
                                <div style={{ fontWeight: 600, marginTop: '2px' }}>
                                    {viewDetails.protocols || 'http,https'}
                                </div>
                            </div>
                            <div>
                                <span style={{ color: '#64748b', fontSize: '12px' }}>Allowed Domains (vHost)</span>
                                <div style={{ fontWeight: 600, marginTop: '2px', display: 'flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}>
                                    {viewDetails.allowed_domains && viewDetails.allowed_domains !== '*' ? (
                                        viewDetails.allowed_domains.split(',').map((d, i) => (
                                            <span key={i} style={{
                                                fontSize: '11.5px',
                                                padding: '2px 8px',
                                                borderRadius: '6px',
                                                backgroundColor: '#eff6ff',
                                                color: '#1d4ed8',
                                                fontFamily: 'monospace',
                                                fontWeight: 600,
                                                border: '1px solid #bfdbfe'
                                            }}>
                                                🌐 {d.trim()}
                                            </span>
                                        ))
                                    ) : (
                                        <span style={{ color: '#64748b', fontStyle: 'italic' }}>* (Any Domain)</span>
                                    )}
                                </div>
                            </div>
                            <div>
                                <span style={{ color: '#64748b', fontSize: '12px' }}>Allowed IPs</span>
                                <div style={{ fontWeight: 600, marginTop: '2px' }}>
                                    {viewDetails.allowed_ips || '*'}
                                </div>
                            </div>
                        </div>

                        {(() => {
                            const rawDomain = (viewDetails.allowed_domains && viewDetails.allowed_domains !== '*')
                                ? viewDetails.allowed_domains.split(',')[0].trim()
                                : '';
                            const hostHeader = rawDomain && !rawDomain.startsWith('*') ? `-H "Host: ${rawDomain}" ` : '';
                            const authHeader = !viewDetails.is_public ? '-H "X-API-Key: YOUR_CONSUMER_KEY" ' : '';
                            const curlCommand = `curl -X ${viewDetails.method} ${hostHeader}${authHeader}"${getGatewayBaseUrl()}${viewDetails.path}"`;

                            return (
                                <div style={{ marginTop: '6px' }}>
                                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                                        <span style={{ color: '#64748b', fontSize: '12px' }}>Example cURL Request</span>
                                        <button
                                            onClick={() => copyToClipboard(curlCommand)}
                                            style={{
                                                border: 'none',
                                                background: 'transparent',
                                                color: copiedCurl ? '#16a34a' : '#2563eb',
                                                fontSize: '12px',
                                                cursor: 'pointer',
                                                display: 'flex',
                                                alignItems: 'center',
                                                gap: '4px'
                                            }}
                                        >
                                            {copiedCurl ? <IconCheck width={13} height={13} /> : <IconCopy width={13} height={13} />}
                                            <span>{copiedCurl ? 'Copied' : 'Copy cURL'}</span>
                                        </button>
                                    </div>
                                    <pre style={{
                                        backgroundColor: '#0f172a',
                                        color: '#38bdf8',
                                        padding: '10px 14px',
                                        borderRadius: '6px',
                                        fontSize: '12px',
                                        overflowX: 'auto',
                                        marginTop: '4px'
                                    }}>
                                        {curlCommand}
                                    </pre>
                                </div>
                            );
                        })()}

                        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '10px' }}>
                            <button className="btn-header-refresh" onClick={() => setViewDetails(null)}>
                                Close
                            </button>
                        </div>
                    </div>
                </Modal>
            )}
        </div>
    );
}
