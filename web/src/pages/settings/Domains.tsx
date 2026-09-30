import React, { useState, useEffect } from 'react';
import { apiClient } from '../../api/apiClient';
import { useToast } from '../../components/Toast';
import { Modal } from '../../components/Modal';
import {
    IconGlobe,
    IconShield,
    IconPlus,
    IconRefresh,
    IconSearch,
    IconTrash,
    IconCheck,
    IconCopy
} from '../../components/Icons';

interface GatewayDomain {
    id: string;
    domain_name: string;
    display_name: string;
    is_default: boolean;
    environment: string;
    default_allowed_ips?: string;
    description: string | null;
    created_at: string;
}

export default function Domains() {
    const [domains, setDomains] = useState<GatewayDomain[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [searchQuery, setSearchQuery] = useState('');
    const [envFilter, setEnvFilter] = useState('ALL');
    const [copiedDomain, setCopiedDomain] = useState<string | null>(null);

    const [form, setForm] = useState({
        domain_name: '',
        display_name: '',
        environment: 'production',
        is_default: false,
        default_allowed_ips: '*',
        description: '',
    });

    const toast = useToast();

    const fetchDomains = async () => {
        try {
            setIsLoading(true);
            const res = await apiClient.get<any>('/domains');
            const list = Array.isArray(res.data) ? res.data : (res.data?.data || []);
            setDomains(list);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load domains');
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        fetchDomains();
    }, []);

    const handleCreate = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!form.domain_name.trim()) {
            toast.error('Domain name is required');
            return;
        }

        try {
            setIsSubmitting(true);
            await apiClient.post('/domains', form);
            toast.success('Gateway domain registered successfully');
            setIsModalOpen(false);
            setForm({
                domain_name: '',
                display_name: '',
                environment: 'production',
                is_default: false,
                default_allowed_ips: '*',
                description: '',
            });
            fetchDomains();
        } catch (err: any) {
            toast.error(err.message || 'Failed to create domain');
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleDelete = async (id: string, domainName: string) => {
        if (!window.confirm(`Are you sure you want to remove domain "${domainName}"? Routes bound to this domain may be affected.`)) {
            return;
        }

        try {
            await apiClient.delete(`/domains/${id}`);
            toast.success(`Domain "${domainName}" removed successfully`);
            fetchDomains();
        } catch (err: any) {
            toast.error(err.message || 'Failed to delete domain');
        }
    };

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        setCopiedDomain(text);
        setTimeout(() => setCopiedDomain(null), 2000);
    };

    const filteredDomains = domains.filter((d) => {
        const matchesQuery =
            d.domain_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            (d.display_name && d.display_name.toLowerCase().includes(searchQuery.toLowerCase())) ||
            (d.description && d.description.toLowerCase().includes(searchQuery.toLowerCase()));
        const matchesEnv = envFilter === 'ALL' || d.environment.toLowerCase() === envFilter.toLowerCase();
        return matchesQuery && matchesEnv;
    });

    const prodCount = domains.filter((d) => d.environment.toLowerCase() === 'production').length;
    const defaultDomain = domains.find((d) => d.is_default);

    const getEnvBadge = (env: string) => {
        switch (env?.toLowerCase()) {
            case 'production':
                return { bg: '#dcfce7', color: '#15803d', border: '#bbf7d0', label: 'Production' };
            case 'staging':
                return { bg: '#fef3c7', color: '#b45309', border: '#fde68a', label: 'Staging' };
            case 'internal':
                return { bg: '#f3e8ff', color: '#7e22ce', border: '#e9d5ff', label: 'Internal / VPN' };
            case 'development':
                return { bg: '#eff6ff', color: '#1d4ed8', border: '#bfdbfe', label: 'Development' };
            default:
                return { bg: '#f1f5f9', color: '#475569', border: '#e2e8f0', label: env || 'Standard' };
        }
    };

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Console Settings</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Domains &amp; TLS</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Domains &amp; TLS Management</h1>
                    <p className="overview-subtitle">
                        Configure custom hostnames, edge TLS/SSL encryption certificates, and host header routing rules for Gateway traffic.
                    </p>
                </div>

                <div className="overview-header-actions">
                    <button
                        className="btn-header-refresh"
                        onClick={fetchDomains}
                        disabled={isLoading}
                        title="Reload domains list"
                    >
                        <IconRefresh width={14} height={14} />
                        <span>Refresh</span>
                    </button>
                    <button
                        className="btn-header-new-route"
                        onClick={() => setIsModalOpen(true)}
                    >
                        <IconPlus width={15} height={15} />
                        <span>Register Domain</span>
                    </button>
                </div>
            </div>

            {/* Summary Metrics Cards */}
            <div
                className="overview-metrics-grid"
                style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
                    gap: '16px'
                }}
            >
                <div
                    className="overview-metric-card"
                    style={{
                        background: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '12px',
                        padding: '20px',
                        boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.04)',
                        display: 'flex',
                        flexDirection: 'column',
                        justifyContent: 'space-between',
                        minHeight: '128px'
                    }}
                >
                    <div className="metric-card-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span className="metric-card-label" style={{ fontSize: '13.5px', fontWeight: 500, color: '#64748b' }}>TOTAL DOMAINS</span>
                        <div className="metric-card-icon-badge" style={{ width: '32px', height: '32px', borderRadius: '50%', background: '#eff6ff', color: '#2563eb', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                            <IconGlobe width={16} height={16} />
                        </div>
                    </div>
                    <div className="metric-card-body" style={{ marginTop: '12px' }}>
                        <div className="metric-card-number" style={{ fontSize: '30px', fontWeight: 700, color: '#0f172a', letterSpacing: '-0.6px', lineHeight: 1.1 }}>
                            {domains.length}
                        </div>
                        <div className="metric-card-trend" style={{ fontSize: '12.5px', fontWeight: 500, marginTop: '6px', color: '#64748b' }}>
                            Registered gateway hostnames
                        </div>
                    </div>
                </div>

                <div
                    className="overview-metric-card"
                    style={{
                        background: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '12px',
                        padding: '20px',
                        boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.04)',
                        display: 'flex',
                        flexDirection: 'column',
                        justifyContent: 'space-between',
                        minHeight: '128px'
                    }}
                >
                    <div className="metric-card-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span className="metric-card-label" style={{ fontSize: '13.5px', fontWeight: 500, color: '#64748b' }}>PRODUCTION HOSTS</span>
                        <div className="metric-card-icon-badge" style={{ width: '32px', height: '32px', borderRadius: '50%', background: '#f0fdf4', color: '#16a34a', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                            <IconShield width={16} height={16} />
                        </div>
                    </div>
                    <div className="metric-card-body" style={{ marginTop: '12px' }}>
                        <div className="metric-card-number" style={{ fontSize: '30px', fontWeight: 700, color: '#0f172a', letterSpacing: '-0.6px', lineHeight: 1.1 }}>
                            {prodCount}
                        </div>
                        <div className="metric-card-trend" style={{ fontSize: '12.5px', fontWeight: 500, marginTop: '6px', color: '#16a34a' }}>
                            Active production endpoints
                        </div>
                    </div>
                </div>

                <div
                    className="overview-metric-card"
                    style={{
                        background: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '12px',
                        padding: '20px',
                        boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.04)',
                        display: 'flex',
                        flexDirection: 'column',
                        justifyContent: 'space-between',
                        minHeight: '128px'
                    }}
                >
                    <div className="metric-card-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span className="metric-card-label" style={{ fontSize: '13.5px', fontWeight: 500, color: '#64748b' }}>EDGE TLS / SSL</span>
                        <div className="metric-card-icon-badge" style={{ width: '32px', height: '32px', borderRadius: '50%', background: '#ecfdf5', color: '#059669', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                            <IconShield width={16} height={16} />
                        </div>
                    </div>
                    <div className="metric-card-body" style={{ marginTop: '12px' }}>
                        <div className="metric-card-number" style={{ fontSize: '20px', fontWeight: 700, color: '#059669', paddingTop: '6px', lineHeight: 1.1 }}>
                            TLS 1.3 Active
                        </div>
                        <div className="metric-card-trend" style={{ fontSize: '12.5px', fontWeight: 500, marginTop: '6px', color: '#64748b' }}>
                            Automated HTTPS edge termination
                        </div>
                    </div>
                </div>

                <div
                    className="overview-metric-card"
                    style={{
                        background: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '12px',
                        padding: '20px',
                        boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.04)',
                        display: 'flex',
                        flexDirection: 'column',
                        justifyContent: 'space-between',
                        minHeight: '128px'
                    }}
                >
                    <div className="metric-card-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span className="metric-card-label" style={{ fontSize: '13.5px', fontWeight: 500, color: '#64748b' }}>DEFAULT ROUTE HOST</span>
                        <div className="metric-card-icon-badge" style={{ width: '32px', height: '32px', borderRadius: '50%', background: '#faf5ff', color: '#9333ea', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                            <IconGlobe width={16} height={16} />
                        </div>
                    </div>
                    <div className="metric-card-body" style={{ marginTop: '12px' }}>
                        <div
                            className="metric-card-number"
                            style={{
                                fontSize: '16px',
                                fontWeight: 700,
                                color: defaultDomain ? '#0f172a' : '#64748b',
                                paddingTop: '8px',
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                                lineHeight: 1.1
                            }}
                            title={defaultDomain?.domain_name || 'All Hosts (*) fallback'}
                        >
                            {defaultDomain ? defaultDomain.domain_name : 'All Wildcard Hosts (*)'}
                        </div>
                        <div className="metric-card-trend" style={{ fontSize: '12.5px', fontWeight: 500, marginTop: '6px', color: '#64748b' }}>
                            Primary fallback routing header
                        </div>
                    </div>
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
                boxShadow: '0 1px 2px rgba(0, 0, 0, 0.03)'
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flex: 1, minWidth: '280px' }}>
                    <div style={{
                        position: 'relative',
                        flex: 1,
                        maxWidth: '420px',
                        display: 'flex',
                        alignItems: 'center'
                    }}>
                        <IconSearch
                            width={16}
                            height={16}
                            style={{ position: 'absolute', left: '12px', color: '#94a3b8' }}
                        />
                        <input
                            type="text"
                            placeholder="Search by domain, hostname, or description..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            style={{
                                width: '100%',
                                height: '38px',
                                padding: '0 12px 0 36px',
                                fontSize: '13.5px',
                                border: '1px solid #e2e8f0',
                                borderRadius: '8px',
                                backgroundColor: '#f8fafc',
                                color: '#0f172a',
                                outline: 'none',
                                transition: 'all 0.2s'
                            }}
                        />
                    </div>

                    <select
                        value={envFilter}
                        onChange={(e) => setEnvFilter(e.target.value)}
                        style={{
                            height: '38px',
                            padding: '0 12px',
                            fontSize: '13px',
                            border: '1px solid #e2e8f0',
                            borderRadius: '8px',
                            backgroundColor: '#ffffff',
                            color: '#334155',
                            outline: 'none',
                            fontWeight: 500,
                            cursor: 'pointer'
                        }}
                    >
                        <option value="ALL">All Environments</option>
                        <option value="production">Production</option>
                        <option value="staging">Staging</option>
                        <option value="internal">Internal / VPN</option>
                        <option value="development">Development</option>
                    </select>
                </div>

                <div style={{ fontSize: '13px', color: '#64748b', fontWeight: 500 }}>
                    Showing <strong style={{ color: '#0f172a' }}>{filteredDomains.length}</strong> of {domains.length} domains
                </div>
            </div>

            {/* Domains Table Container */}
            <div style={{
                background: '#ffffff',
                border: '1px solid #e2e8f0',
                borderRadius: '12px',
                overflow: 'hidden',
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.04)'
            }}>
                {isLoading ? (
                    <div style={{ padding: '60px', textAlign: 'center' }}>
                        <div className="spinner" style={{ margin: '0 auto' }}></div>
                        <div style={{ marginTop: '16px', color: '#64748b', fontSize: '14px' }}>Loading registered domains...</div>
                    </div>
                ) : filteredDomains.length === 0 ? (
                    <div style={{ padding: '60px 24px', textAlign: 'center' }}>
                        <div style={{
                            width: '56px',
                            height: '56px',
                            borderRadius: '50%',
                            background: '#eff6ff',
                            color: '#2563eb',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            margin: '0 auto 16px'
                        }}>
                            <IconGlobe width={28} height={28} />
                        </div>
                        <h3 style={{ fontSize: '17px', fontWeight: 600, color: '#0f172a', margin: '0 0 6px' }}>
                            {domains.length === 0 ? 'No Gateway Domains Configured' : 'No Matching Domains'}
                        </h3>
                        <p style={{ fontSize: '13.5px', color: '#64748b', margin: '0 0 20px', maxWidth: '440px', marginLeft: 'auto', marginRight: 'auto' }}>
                            {domains.length === 0
                                ? 'Register your custom hostnames to enable multi-tenant host header matching and edge TLS/SSL routing.'
                                : 'No domains match your current search query or environment filter.'}
                        </p>
                        {domains.length === 0 && (
                            <button
                                className="btn-header-new-route"
                                onClick={() => setIsModalOpen(true)}
                                style={{ margin: '0 auto' }}
                            >
                                <IconPlus width={15} height={15} />
                                <span>Register First Domain</span>
                            </button>
                        )}
                    </div>
                ) : (
                    <div style={{ overflowX: 'auto' }}>
                        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13.5px', textAlign: 'left' }}>
                            <thead>
                                <tr style={{
                                    borderBottom: '1px solid #e2e8f0',
                                    background: '#f8fafc',
                                    color: '#64748b',
                                    fontSize: '12px',
                                    fontWeight: 600,
                                    textTransform: 'uppercase',
                                    letterSpacing: '0.5px'
                                }}>
                                    <th style={{ padding: '14px 20px' }}>Domain / Hostname</th>
                                    <th style={{ padding: '14px 20px' }}>Environment</th>
                                    <th style={{ padding: '14px 20px' }}>TLS / HTTPS</th>
                                    <th style={{ padding: '14px 20px' }}>Allowed Client IPs</th>
                                    <th style={{ padding: '14px 20px' }}>Routing Role</th>
                                    <th style={{ padding: '14px 20px' }}>Description</th>
                                    <th style={{ padding: '14px 20px', textAlign: 'right' }}>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredDomains.map((dom) => {
                                    const envBadge = getEnvBadge(dom.environment);
                                    return (
                                        <tr
                                            key={dom.id}
                                            style={{
                                                borderBottom: '1px solid #f1f5f9',
                                                transition: 'background-color 0.15s'
                                            }}
                                            onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#fafafa')}
                                            onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = 'transparent')}
                                        >
                                            {/* Domain Name */}
                                            <td style={{ padding: '16px 20px' }}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                                    <div style={{
                                                        width: '32px',
                                                        height: '32px',
                                                        borderRadius: '8px',
                                                        background: '#eff6ff',
                                                        color: '#2563eb',
                                                        display: 'flex',
                                                        alignItems: 'center',
                                                        justifyContent: 'center',
                                                        flexShrink: 0
                                                    }}>
                                                        <IconGlobe width={18} height={18} />
                                                    </div>
                                                    <div>
                                                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                            <span style={{ fontWeight: 600, color: '#0f172a', fontFamily: 'monospace', fontSize: '13.5px' }}>
                                                                {dom.domain_name}
                                                            </span>
                                                            <button
                                                                onClick={() => copyToClipboard(dom.domain_name)}
                                                                title="Copy hostname"
                                                                style={{
                                                                    border: 'none',
                                                                    background: 'transparent',
                                                                    color: copiedDomain === dom.domain_name ? '#16a34a' : '#94a3b8',
                                                                    cursor: 'pointer',
                                                                    padding: '2px',
                                                                    display: 'inline-flex',
                                                                    alignItems: 'center'
                                                                }}
                                                            >
                                                                {copiedDomain === dom.domain_name ? (
                                                                    <IconCheck width={13} height={13} />
                                                                ) : (
                                                                    <IconCopy width={13} height={13} />
                                                                )}
                                                            </button>
                                                        </div>
                                                        <div style={{ fontSize: '12px', color: '#64748b', marginTop: '2px' }}>
                                                            {dom.display_name || 'Primary Host'}
                                                        </div>
                                                    </div>
                                                </div>
                                            </td>

                                            {/* Environment Badge */}
                                            <td style={{ padding: '16px 20px' }}>
                                                <span style={{
                                                    fontSize: '11.5px',
                                                    fontWeight: 600,
                                                    padding: '3px 9px',
                                                    borderRadius: '6px',
                                                    background: envBadge.bg,
                                                    color: envBadge.color,
                                                    border: `1px solid ${envBadge.border}`,
                                                    display: 'inline-flex',
                                                    alignItems: 'center'
                                                }}>
                                                    {envBadge.label}
                                                </span>
                                            </td>

                                            {/* TLS / HTTPS */}
                                            <td style={{ padding: '16px 20px' }}>
                                                <span style={{
                                                    fontSize: '11.5px',
                                                    fontWeight: 600,
                                                    padding: '3px 9px',
                                                    borderRadius: '6px',
                                                    background: '#ecfdf5',
                                                    color: '#059669',
                                                    border: '1px solid #a7f3d0',
                                                    display: 'inline-flex',
                                                    alignItems: 'center',
                                                    gap: '5px'
                                                }} title="TLS 1.3 Active - Automatic SSL termination">
                                                    <IconShield width={13} height={13} />
                                                    <span>Active (HTTPS)</span>
                                                </span>
                                            </td>

                                            {/* Allowed Client IPs */}
                                            <td style={{ padding: '16px 20px' }}>
                                                <span style={{
                                                    fontSize: '12px',
                                                    fontFamily: 'monospace',
                                                    color: '#334155',
                                                    background: '#f1f5f9',
                                                    padding: '3px 8px',
                                                    borderRadius: '6px',
                                                    border: '1px solid #e2e8f0',
                                                    display: 'inline-block'
                                                }}>
                                                    {dom.default_allowed_ips || '* (All IPs)'}
                                                </span>
                                            </td>

                                            {/* Role */}
                                            <td style={{ padding: '16px 20px' }}>
                                                {dom.is_default ? (
                                                    <span style={{
                                                        fontSize: '11.5px',
                                                        fontWeight: 700,
                                                        padding: '3px 9px',
                                                        borderRadius: '6px',
                                                        background: '#f0fdf4',
                                                        color: '#166534',
                                                        border: '1px solid #bbf7d0',
                                                        letterSpacing: '0.3px',
                                                        display: 'inline-flex',
                                                        alignItems: 'center',
                                                        gap: '4px'
                                                    }}>
                                                        <span>★ DEFAULT</span>
                                                    </span>
                                                ) : (
                                                    <span style={{ fontSize: '12.5px', color: '#64748b' }}>
                                                        Secondary
                                                    </span>
                                                )}
                                            </td>

                                            {/* Description */}
                                            <td style={{ padding: '16px 20px', color: '#64748b', fontSize: '13px', maxWidth: '240px' }}>
                                                <div style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={dom.description || ''}>
                                                    {dom.description || '—'}
                                                </div>
                                            </td>

                                            {/* Actions */}
                                            <td style={{ padding: '16px 20px', textAlign: 'right' }}>
                                                <button
                                                    onClick={() => handleDelete(dom.id, dom.domain_name)}
                                                    disabled={dom.is_default && domains.length === 1}
                                                    title={dom.is_default && domains.length === 1 ? 'Cannot remove the sole default domain' : 'Delete domain'}
                                                    style={{
                                                        display: 'inline-flex',
                                                        alignItems: 'center',
                                                        gap: '4px',
                                                        padding: '6px 10px',
                                                        background: '#ffffff',
                                                        color: '#dc2626',
                                                        border: '1px solid #fee2e2',
                                                        borderRadius: '6px',
                                                        fontSize: '12.5px',
                                                        fontWeight: 500,
                                                        cursor: dom.is_default && domains.length === 1 ? 'not-allowed' : 'pointer',
                                                        opacity: dom.is_default && domains.length === 1 ? 0.4 : 1,
                                                        transition: 'all 0.15s'
                                                    }}
                                                    onMouseEnter={(e) => {
                                                        if (!(dom.is_default && domains.length === 1)) {
                                                            e.currentTarget.style.backgroundColor = '#fef2f2';
                                                            e.currentTarget.style.borderColor = '#fca5a5';
                                                        }
                                                    }}
                                                    onMouseLeave={(e) => {
                                                        e.currentTarget.style.backgroundColor = '#ffffff';
                                                        e.currentTarget.style.borderColor = '#fee2e2';
                                                    }}
                                                >
                                                    <IconTrash width={14} height={14} />
                                                    <span>Delete</span>
                                                </button>
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* Register Domain Modal */}
            <Modal
                isOpen={isModalOpen}
                onClose={() => setIsModalOpen(false)}
                title="Register Gateway Domain"
                size="md"
            >
                <form onSubmit={handleCreate} style={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
                    <div>
                        <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: '#334155', marginBottom: '6px' }}>
                            Domain Name / Host Header *
                        </label>
                        <input
                            type="text"
                            placeholder="e.g. api.apirout.io or gateway.internal.corp"
                            value={form.domain_name}
                            onChange={(e) => setForm({ ...form, domain_name: e.target.value })}
                            required
                            style={{
                                width: '100%',
                                height: '40px',
                                padding: '0 12px',
                                fontSize: '13.5px',
                                border: '1px solid #cbd5e1',
                                borderRadius: '8px',
                                outline: 'none'
                            }}
                        />
                        <div style={{ fontSize: '12px', color: '#64748b', marginTop: '4px' }}>
                            The exact HTTP Host header matched by incoming requests to the Data Plane Gateway.
                        </div>
                    </div>

                    <div>
                        <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: '#334155', marginBottom: '6px' }}>
                            Display Name / Label
                        </label>
                        <input
                            type="text"
                            placeholder="e.g. Public Production API Gateway"
                            value={form.display_name}
                            onChange={(e) => setForm({ ...form, display_name: e.target.value })}
                            style={{
                                width: '100%',
                                height: '40px',
                                padding: '0 12px',
                                fontSize: '13.5px',
                                border: '1px solid #cbd5e1',
                                borderRadius: '8px',
                                outline: 'none'
                            }}
                        />
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
                        <div>
                            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: '#334155', marginBottom: '6px' }}>
                                Environment
                            </label>
                            <select
                                value={form.environment}
                                onChange={(e) => setForm({ ...form, environment: e.target.value })}
                                style={{
                                    width: '100%',
                                    height: '40px',
                                    padding: '0 12px',
                                    fontSize: '13.5px',
                                    border: '1px solid #cbd5e1',
                                    borderRadius: '8px',
                                    backgroundColor: '#ffffff',
                                    outline: 'none',
                                    cursor: 'pointer'
                                }}
                            >
                                <option value="production">Production</option>
                                <option value="staging">Staging</option>
                                <option value="internal">Internal / VPN</option>
                                <option value="development">Development</option>
                            </select>
                        </div>

                        <div>
                            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: '#334155', marginBottom: '6px' }}>
                                Allowed Client IPs (CIDR)
                            </label>
                            <input
                                type="text"
                                placeholder="* (All) or 10.0.0.0/8, 192.168.1.0/24"
                                value={form.default_allowed_ips}
                                onChange={(e) => setForm({ ...form, default_allowed_ips: e.target.value })}
                                style={{
                                    width: '100%',
                                    height: '40px',
                                    padding: '0 12px',
                                    fontSize: '13.5px',
                                    border: '1px solid #cbd5e1',
                                    borderRadius: '8px',
                                    outline: 'none'
                                }}
                            />
                        </div>
                    </div>

                    <div style={{
                        padding: '12px 14px',
                        background: '#f8fafc',
                        border: '1px solid #e2e8f0',
                        borderRadius: '8px',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '10px',
                        cursor: 'pointer'
                    }} onClick={() => setForm(f => ({ ...f, is_default: !f.is_default }))}>
                        <input
                            type="checkbox"
                            checked={form.is_default}
                            onChange={(e) => setForm({ ...form, is_default: e.target.checked })}
                            style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                        />
                        <div>
                            <div style={{ fontSize: '13px', fontWeight: 600, color: '#0f172a' }}>
                                Set as Default Gateway Domain
                            </div>
                            <div style={{ fontSize: '12px', color: '#64748b' }}>
                                Incoming traffic without matching host headers will route to this domain.
                            </div>
                        </div>
                    </div>

                    <div>
                        <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: '#334155', marginBottom: '6px' }}>
                            Description (Optional)
                        </label>
                        <textarea
                            rows={3}
                            placeholder="Operational notes or internal purpose of this gateway host..."
                            value={form.description}
                            onChange={(e) => setForm({ ...form, description: e.target.value })}
                            style={{
                                width: '100%',
                                padding: '10px 12px',
                                fontSize: '13.5px',
                                border: '1px solid #cbd5e1',
                                borderRadius: '8px',
                                outline: 'none',
                                resize: 'vertical'
                            }}
                        />
                    </div>

                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '10px', borderTop: '1px solid #f1f5f9', paddingTop: '16px' }}>
                        <button
                            type="button"
                            className="btn-header-refresh"
                            onClick={() => setIsModalOpen(false)}
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            className="btn-header-new-route"
                            disabled={isSubmitting}
                        >
                            {isSubmitting ? 'Registering...' : 'Register Domain'}
                        </button>
                    </div>
                </form>
            </Modal>
        </div>
    );
}
