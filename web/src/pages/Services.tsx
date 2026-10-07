import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { gatewayApi, GatewayService } from '../api/gatewayApi';
import { useToast } from '../components/Toast';
import { Modal } from '../components/Modal';
import {
    IconServices,
    IconSearch,
    IconRefresh,
    IconPlus,
    IconPencil,
    IconTrash,
    IconDatabase,
    IconLayers,
    IconServer,
    IconRoutes
} from '../components/Icons';

export default function ServicesPage() {
    const navigate = useNavigate();
    const [services, setServices] = useState<GatewayService[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const [protocolFilter, setProtocolFilter] = useState('all');

    // Routes drawer for service
    const [selectedServiceForRoutes, setSelectedServiceForRoutes] = useState<GatewayService | null>(null);
    const [serviceRoutes, setServiceRoutes] = useState<any[]>([]);
    const [isRoutesDrawerOpen, setIsRoutesDrawerOpen] = useState(false);
    const [isRoutesLoading, setIsRoutesLoading] = useState(false);

    const inputRef = useRef<HTMLInputElement>(null);
    const toast = useToast();

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setIsLoading(true);
        try {
            const servicesRes = await gatewayApi.getServices().catch(() => ({ data: [] }));
            setServices(servicesRes.data || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load services');
        } finally {
            setIsLoading(false);
        }
    };

    const handleOpenCreateService = () => {
        navigate('/services/new');
    };

    const handleOpenEditService = (service: GatewayService) => {
        navigate(`/services/${service.id}/edit`);
    };

    const handleDeleteService = async (service: GatewayService) => {
        if (!window.confirm(`Are you sure you want to delete service "${service.name}"?`)) return;
        try {
            await gatewayApi.deleteService(service.id);
            toast.success(`Service ${service.name} deleted`);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to delete service');
        }
    };

    const handleViewRoutes = async (service: GatewayService) => {
        setSelectedServiceForRoutes(service);
        setIsRoutesLoading(true);
        setIsRoutesDrawerOpen(true);
        try {
            const res = await gatewayApi.getServiceRoutes(service.id);
            setServiceRoutes(res.data || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load routes');
        } finally {
            setIsRoutesLoading(false);
        }
    };

    // Filter services
    const filteredServices = services.filter(s => {
        const matchesSearch =
            s.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            (s.description && s.description.toLowerCase().includes(searchTerm.toLowerCase())) ||
            (s.host && s.host.toLowerCase().includes(searchTerm.toLowerCase())) ||
            (s.upstream_name && s.upstream_name.toLowerCase().includes(searchTerm.toLowerCase()));

        const matchesProtocol =
            protocolFilter === 'all' || s.protocol === protocolFilter;

        return matchesSearch && matchesProtocol;
    });

    const getProtocolBadge = (service: GatewayService) => {
        if (service.upstream_id) {
            return { label: 'Upstream Pool', bg: '#faf5ff', color: '#9333ea', icon: <IconLayers width={12} height={12} /> };
        }
        if (service.protocol === 'https') {
            return { label: 'HTTPS', bg: '#eff6ff', color: '#2563eb', icon: <IconServer width={12} height={12} /> };
        }
        return { label: 'HTTP', bg: '#f1f5f9', color: '#475569', icon: <IconServer width={12} height={12} /> };
    };

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Gateway</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Services</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Gateway Services</h1>
                    <p className="overview-subtitle">
                        Logical backend services. Each service represents an upstream application, host, or load-balancing pool.
                    </p>
                </div>

                <div className="overview-header-actions">
                    <button className="btn-header-refresh" onClick={loadData} title="Reload data">
                        <IconRefresh width={14} height={14} />
                        <span>Refresh</span>
                    </button>
                    <button className="btn-header-new-route" onClick={handleOpenCreateService}>
                        <IconPlus width={15} height={15} />
                        <span>New Service</span>
                    </button>
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
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flex: '1 1 360px', flexWrap: 'wrap' }}>
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
                                    placeholder="Search services by name, host, or pool... (Ctrl+K)"
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
                                value={protocolFilter}
                                onChange={e => setProtocolFilter(e.target.value)}
                            >
                                <option value="all">All Targets</option>
                                <option value="http">Direct HTTP</option>
                                <option value="https">Direct HTTPS</option>
                                <option value="database">Database (Data Source)</option>
                            </select>
                        </div>

                        <div style={{ fontSize: '13px', color: '#64748b' }}>
                            Showing <strong>{filteredServices.length}</strong> of {services.length} services
                        </div>
                    </div>

                    {/* Table */}
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
                                <p style={{ color: '#64748b', fontSize: '14px' }}>Loading services...</p>
                            </div>
                        ) : filteredServices.length === 0 ? (
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
                                    <IconServices width={24} height={24} />
                                </div>
                                <h3 style={{ fontSize: '16px', fontWeight: 600, color: '#0f172a', marginBottom: '6px' }}>
                                    {services.length === 0 ? 'No Services Configured' : 'No Matching Services'}
                                </h3>
                                <p style={{ fontSize: '13.5px', color: '#64748b', maxWidth: '420px', margin: '0 auto 20px auto' }}>
                                    {services.length === 0
                                        ? 'Create a Service pointing directly to a host, Upstream pool, or Database connection.'
                                        : 'Try modifying your search or protocol filters.'}
                                </p>
                                {services.length === 0 && (
                                    <button className="btn-header-new-route" onClick={handleOpenCreateService}>
                                        Create Service
                                    </button>
                                )}
                            </div>
                        ) : (
                            <div style={{ overflowX: 'auto' }}>
                                <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '13.5px' }}>
                                    <thead>
                                        <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Service Name</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Target Resolution</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Protocol / Type</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Linked Routes</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Status</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569', textAlign: 'right' }}>Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {filteredServices.map((service, idx) => {
                                            const badge = getProtocolBadge(service);
                                            return (
                                                <tr
                                                    key={service.id}
                                                    style={{
                                                        borderTop: idx > 0 ? '1px solid #f1f5f9' : 'none',
                                                        backgroundColor: idx % 2 === 0 ? '#ffffff' : '#fafafa',
                                                        transition: 'background 100ms ease'
                                                    }}
                                                >
                                                    <td style={{ padding: '12px 20px' }}>
                                                        <div style={{ display: 'flex', flexDirection: 'column' }}>
                                                            <span style={{ fontWeight: 600, color: '#0f172a', fontSize: '14px' }}>
                                                                {service.name}
                                                            </span>
                                                            {service.description && (
                                                                <span style={{ fontSize: '12px', color: '#64748b', marginTop: '2px' }}>
                                                                    {service.description}
                                                                </span>
                                                            )}
                                                        </div>
                                                    </td>

                                                    <td style={{ padding: '12px 20px', color: '#334155' }}>
                                                        {service.upstream_id ? (
                                                            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                                <IconLayers width={14} height={14} style={{ color: '#9333ea' }} />
                                                                <span style={{ fontWeight: 500 }}>
                                                                    Upstream: <strong>{service.upstream_name || service.upstream_id}</strong>
                                                                </span>
                                                            </div>
                                                        ) : service.connection_id || service.protocol === 'database' ? (
                                                            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                                <IconDatabase width={14} height={14} style={{ color: '#16a34a' }} />
                                                                <span style={{ fontWeight: 500 }}>
                                                                    DB: <strong>{service.connection_name || service.connection_id}</strong>
                                                                </span>
                                                            </div>
                                                        ) : (
                                                            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                                <code style={{
                                                                    fontFamily: 'monospace',
                                                                    fontSize: '12.5px',
                                                                    backgroundColor: '#f1f5f9',
                                                                    padding: '2px 6px',
                                                                    borderRadius: '4px',
                                                                    color: '#0f172a'
                                                                }}>
                                                                    {service.protocol}://{service.host}:{service.port}{service.path || ''}
                                                                </code>
                                                            </div>
                                                        )}
                                                    </td>

                                                    <td style={{ padding: '12px 20px' }}>
                                                        <span style={{
                                                            display: 'inline-flex',
                                                            alignItems: 'center',
                                                            gap: '4px',
                                                            fontSize: '11.5px',
                                                            fontWeight: 600,
                                                            padding: '3px 8px',
                                                            borderRadius: '6px',
                                                            backgroundColor: badge.bg,
                                                            color: badge.color
                                                        }}>
                                                            {badge.icon}
                                                            <span>{badge.label}</span>
                                                        </span>
                                                    </td>

                                                    <td style={{ padding: '12px 20px' }}>
                                                        <button
                                                            onClick={() => handleViewRoutes(service)}
                                                            style={{
                                                                display: 'inline-flex',
                                                                alignItems: 'center',
                                                                gap: '6px',
                                                                border: '1px solid #e2e8f0',
                                                                borderRadius: '6px',
                                                                padding: '4px 10px',
                                                                backgroundColor: '#ffffff',
                                                                color: '#334155',
                                                                fontSize: '12px',
                                                                fontWeight: 500,
                                                                cursor: 'pointer'
                                                            }}
                                                        >
                                                            <IconRoutes width={13} height={13} style={{ color: '#2563eb' }} />
                                                            <span>{service.routes_count || 0} Routes</span>
                                                        </button>
                                                    </td>

                                                    <td style={{ padding: '12px 20px' }}>
                                                        <span style={{
                                                            fontSize: '11px',
                                                            fontWeight: 600,
                                                            padding: '2px 8px',
                                                            borderRadius: '12px',
                                                            backgroundColor: service.enabled ? '#dcfce7' : '#fee2e2',
                                                            color: service.enabled ? '#16a34a' : '#dc2626'
                                                        }}>
                                                            {service.enabled ? 'Active' : 'Disabled'}
                                                        </span>
                                                    </td>

                                                    <td style={{ padding: '12px 20px', textAlign: 'right' }}>
                                                        <div style={{ display: 'inline-flex', gap: '8px' }}>
                                                            <button
                                                                onClick={() => handleViewRoutes(service)}
                                                                title="View Linked Routes"
                                                                style={{
                                                                    border: '1px solid #e2e8f0',
                                                                    backgroundColor: '#ffffff',
                                                                    padding: '6px',
                                                                    borderRadius: '6px',
                                                                    color: '#2563eb',
                                                                    cursor: 'pointer'
                                                                }}
                                                            >
                                                                <IconRoutes width={14} height={14} />
                                                            </button>
                                                            <button
                                                                onClick={() => handleOpenEditService(service)}
                                                                title="Edit Service"
                                                                style={{
                                                                    border: '1px solid #e2e8f0',
                                                                    backgroundColor: '#ffffff',
                                                                    padding: '6px',
                                                                    borderRadius: '6px',
                                                                    color: '#475569',
                                                                    cursor: 'pointer'
                                                                }}
                                                            >
                                                                <IconPencil width={14} height={14} />
                                                            </button>
                                                            <button
                                                                onClick={() => handleDeleteService(service)}
                                                                title="Delete Service"
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


            {/* DRAWER / MODAL: LINKED ROUTES FOR SERVICE */}
            {/* ========================================================= */}
            {selectedServiceForRoutes && (
                <Modal
                    isOpen={isRoutesDrawerOpen}
                    onClose={() => setIsRoutesDrawerOpen(false)}
                    title={`Routes linked to Service: ${selectedServiceForRoutes.name}`}
                >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                        <p style={{ fontSize: '13px', color: '#64748b', margin: 0 }}>
                            HTTP requests matching these routes are routed to <strong>{selectedServiceForRoutes.name}</strong>.
                        </p>

                        {isRoutesLoading ? (
                            <div style={{ padding: '30px', textAlign: 'center' }}>
                                <div className="spinner" style={{ margin: '0 auto 8px auto' }}></div>
                                <span style={{ fontSize: '13px', color: '#64748b' }}>Loading routes...</span>
                            </div>
                        ) : serviceRoutes.length === 0 ? (
                            <div style={{
                                padding: '32px 16px',
                                textAlign: 'center',
                                backgroundColor: '#f8fafc',
                                borderRadius: '8px',
                                border: '1px dashed #cbd5e1'
                            }}>
                                <IconRoutes width={28} height={28} style={{ color: '#94a3b8', margin: '0 auto 8px auto' }} />
                                <div style={{ fontSize: '14px', fontWeight: 600, color: '#334155' }}>No Routes Linked</div>
                                <p style={{ fontSize: '12.5px', color: '#64748b', margin: '4px 0 12px 0' }}>
                                    Create a Route in the Routes module that references this Service.
                                </p>
                                <button
                                    className="btn-header-new-route"
                                    onClick={() => { window.location.href = '/routes'; }}
                                >
                                    + Go to Routes
                                </button>
                            </div>
                        ) : (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                {serviceRoutes.map(r => (
                                    <div
                                        key={r.id}
                                        style={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            justifyContent: 'space-between',
                                            padding: '10px 14px',
                                            backgroundColor: '#ffffff',
                                            border: '1px solid #e2e8f0',
                                            borderRadius: '8px',
                                        }}
                                    >
                                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                            <span style={{
                                                fontSize: '11px',
                                                fontWeight: 700,
                                                padding: '2px 6px',
                                                borderRadius: '4px',
                                                fontFamily: 'monospace',
                                                backgroundColor: r.method === 'GET' ? '#dcfce7' : '#dbeafe',
                                                color: r.method === 'GET' ? '#16a34a' : '#2563eb'
                                            }}>
                                                {r.method}
                                            </span>
                                            <code style={{ fontFamily: 'monospace', fontSize: '13px', fontWeight: 600, color: '#0f172a' }}>
                                                {r.path}
                                            </code>
                                            {r.name && (
                                                <span style={{ fontSize: '12px', color: '#64748b' }}>({r.name})</span>
                                            )}
                                        </div>
                                        <span style={{
                                            fontSize: '11px',
                                            fontWeight: 600,
                                            padding: '2px 8px',
                                            borderRadius: '12px',
                                            backgroundColor: r.enabled ? '#dcfce7' : '#fee2e2',
                                            color: r.enabled ? '#16a34a' : '#dc2626'
                                        }}>
                                            {r.enabled ? 'Active' : 'Disabled'}
                                        </span>
                                    </div>
                                ))}
                            </div>
                        )}

                        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '12px' }}>
                            <button className="btn-header-refresh" onClick={() => setIsRoutesDrawerOpen(false)}>
                                Close
                            </button>
                        </div>
                    </div>
                </Modal>
            )}
        </div>
    );
}
