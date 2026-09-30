import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { gatewayApi, Upstream, Target } from '../api/gatewayApi';
import { useToast } from '../components/Toast';
import { Modal } from '../components/Modal';
import { useAuth } from '../auth/AuthProvider';
import {
    IconLayers,
    IconSearch,
    IconRefresh,
    IconPlus,
    IconPencil,
    IconTrash,
    IconServer
} from '../components/Icons';

export default function UpstreamsPage() {
    const navigate = useNavigate();
    const [upstreams, setUpstreams] = useState<Upstream[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const [selectedAlgorithm, setSelectedAlgorithm] = useState<string>('all');

    // Targets management drawer state
    const [selectedUpstreamForTargets, setSelectedUpstreamForTargets] = useState<Upstream | null>(null);
    const [targets, setTargets] = useState<Target[]>([]);
    const [isTargetsLoading, setIsTargetsLoading] = useState(false);
    const [isAddTargetOpen, setIsAddTargetOpen] = useState(false);
    const [targetForm, setTargetForm] = useState({
        target: '',
        weight: 100,
        enabled: true,
        failover: false,
    });
    const [healthCheckingTargetId, setHealthCheckingTargetId] = useState<string | null>(null);
    const [isCheckingAllHealth, setIsCheckingAllHealth] = useState(false);

    const inputRef = useRef<HTMLInputElement>(null);
    const { user, canManageEndpoints } = useAuth();
    const toast = useToast();

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setIsLoading(true);
        try {
            const upstreamsRes = await gatewayApi.getUpstreams().catch(() => ({ data: [] }));
            setUpstreams(upstreamsRes.data || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load upstreams');
        } finally {
            setIsLoading(false);
        }
    };

    const handleOpenCreateUpstream = () => {
        navigate('/upstreams/new');
    };

    const handleOpenEditUpstream = (upstream: Upstream) => {
        navigate(`/upstreams/${upstream.id}/edit`);
    };

    const handleDeleteUpstream = async (upstream: Upstream) => {
        if (!window.confirm(`Are you sure you want to delete upstream pool "${upstream.name}"?`)) return;
        try {
            await gatewayApi.deleteUpstream(upstream.id);
            toast.success(`Upstream ${upstream.name} deleted`);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to delete upstream');
        }
    };

    // Targets Drawer Management
    const handleOpenTargetsDrawer = async (upstream: Upstream) => {
        setSelectedUpstreamForTargets(upstream);
        setIsTargetsLoading(true);
        setIsAddTargetOpen(false);
        try {
            const res = await gatewayApi.getTargets(upstream.id);
            setTargets(res.data || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load targets');
        } finally {
            setIsTargetsLoading(false);
        }
    };

    const handleAddTarget = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!selectedUpstreamForTargets) return;
        if (!targetForm.target.trim()) {
            toast.error('Target instance host:port is required');
            return;
        }

        try {
            await gatewayApi.createTarget(selectedUpstreamForTargets.id, {
                target: targetForm.target.trim(),
                weight: Number(targetForm.weight) || 100,
                enabled: targetForm.enabled,
                failover: targetForm.failover,
            });
            toast.success('Backend target instance added');
            setTargetForm({ target: '', weight: 100, enabled: true, failover: false });
            setIsAddTargetOpen(false);

            // Reload targets
            const res = await gatewayApi.getTargets(selectedUpstreamForTargets.id);
            setTargets(res.data || []);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to add target');
        }
    };

    const handleDeleteTarget = async (targetId: string) => {
        if (!window.confirm('Remove this backend target instance?')) return;
        try {
            await gatewayApi.deleteTarget(targetId);
            toast.success('Target removed');
            if (selectedUpstreamForTargets) {
                const res = await gatewayApi.getTargets(selectedUpstreamForTargets.id);
                setTargets(res.data || []);
            }
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to remove target');
        }
    };

    const handleToggleTarget = async (target: Target) => {
        try {
            await gatewayApi.toggleTarget(target.id, !target.enabled);
            toast.success(`Target ${!target.enabled ? 'enabled' : 'disabled'}`);
            if (selectedUpstreamForTargets) {
                const res = await gatewayApi.getTargets(selectedUpstreamForTargets.id);
                setTargets(res.data || []);
            }
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to toggle target');
        }
    };

    const handleCheckSingleTarget = async (target: Target) => {
        setHealthCheckingTargetId(target.id);
        try {
            const res = await gatewayApi.healthCheckTarget(target.id);
            if (res.health_status === 'healthy') {
                toast.success(`Target is Healthy (${res.latency_ms}ms)`);
            } else {
                toast.error(`Target Unhealthy: ${res.error || 'Failed response'}`);
            }
            if (selectedUpstreamForTargets) {
                const updated = await gatewayApi.getTargets(selectedUpstreamForTargets.id);
                setTargets(updated.data || []);
            }
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Health check error');
        } finally {
            setHealthCheckingTargetId(null);
        }
    };

    const handleCheckPoolHealth = async (upstream: Upstream) => {
        setIsCheckingAllHealth(true);
        try {
            const res = await gatewayApi.healthCheckUpstream(upstream.id);
            toast.success(`Health check completed across ${(res.results || []).length} targets`);
            if (selectedUpstreamForTargets && selectedUpstreamForTargets.id === upstream.id) {
                const updated = await gatewayApi.getTargets(upstream.id);
                setTargets(updated.data || []);
            }
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Pool health check error');
        } finally {
            setIsCheckingAllHealth(false);
        }
    };

    const canManage = user?.role === 'SUPER_ADMIN' || user?.role === 'ADMIN' || canManageEndpoints();

    const filteredUpstreams = upstreams.filter(u => {
        const matchesSearch =
            u.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            (u.targets && u.targets.some(t => t.target.toLowerCase().includes(searchTerm.toLowerCase())));
        const matchesAlgo = selectedAlgorithm === 'all' || u.algorithm === selectedAlgorithm;
        return matchesSearch && matchesAlgo;
    });

    const getAlgorithmBadge = (algo: string) => {
        switch (algo) {
            case 'round-robin':
                return { label: 'Round-Robin', bg: '#eff6ff', color: '#2563eb' };
            case 'least-connections':
                return { label: 'Least-Conn', bg: '#f0fdf4', color: '#16a34a' };
            case 'consistent-hashing':
                return { label: 'Consistent-Hash', bg: '#faf5ff', color: '#9333ea' };
            case 'latency':
                return { label: 'Latency-Based', bg: '#fffbeb', color: '#d97706' };
            default:
                return { label: algo, bg: '#f1f5f9', color: '#475569' };
        }
    };

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Gateway</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Upstreams &amp; Targets</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Gateway Upstreams &amp; Targets</h1>
                    <p className="overview-subtitle">
                        Kong-aligned load balancing pools. Upstreams distribute incoming traffic across Target instances with active health checks.
                    </p>
                </div>

                <div className="overview-header-actions">
                    <button className="btn-header-refresh" onClick={loadData} title="Reload data">
                        <IconRefresh width={14} height={14} />
                        <span>Refresh</span>
                    </button>
                    {canManage && (
                        <button className="btn-header-new-route" onClick={handleOpenCreateUpstream}>
                            <IconPlus width={15} height={15} />
                            <span>New Upstream</span>
                        </button>
                    )}
                </div>
            </div>

            {/* Upstreams & Targets List */}
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
                                    placeholder="Search upstreams or targets... (Ctrl+K)"
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
                                value={selectedAlgorithm}
                                onChange={e => setSelectedAlgorithm(e.target.value)}
                            >
                                <option value="all">All Algorithms</option>
                                <option value="round-robin">Round-Robin</option>
                                <option value="least-connections">Least-Connections</option>
                                <option value="consistent-hashing">Consistent-Hashing</option>
                                <option value="latency">Latency-Based</option>
                            </select>
                        </div>

                        <div style={{ fontSize: '13px', color: '#64748b' }}>
                            Showing <strong>{filteredUpstreams.length}</strong> of {upstreams.length} upstreams
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
                                <p style={{ color: '#64748b', fontSize: '14px' }}>Loading upstreams...</p>
                            </div>
                        ) : filteredUpstreams.length === 0 ? (
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
                                    <IconLayers width={24} height={24} />
                                </div>
                                <h3 style={{ fontSize: '16px', fontWeight: 600, color: '#0f172a', marginBottom: '6px' }}>
                                    {upstreams.length === 0 ? 'No Upstreams Configured' : 'No Matching Upstreams'}
                                </h3>
                                <p style={{ fontSize: '13.5px', color: '#64748b', maxWidth: '420px', margin: '0 auto 20px auto' }}>
                                    {upstreams.length === 0
                                        ? 'Create an Upstream pool to load balance multiple backend instances with active health checking.'
                                        : 'Try modifying your search or algorithm filters.'}
                                </p>
                                {canManage && upstreams.length === 0 && (
                                    <button className="btn-header-new-route" onClick={handleOpenCreateUpstream}>
                                        Create Upstream
                                    </button>
                                )}
                            </div>
                        ) : (
                            <div style={{ overflowX: 'auto' }}>
                                <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '13.5px' }}>
                                    <thead>
                                        <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Upstream Name</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Algorithm</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Targets &amp; Health</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Active Health Check</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Status</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569', textAlign: 'right' }}>Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {filteredUpstreams.map((upstream, idx) => {
                                            const badge = getAlgorithmBadge(upstream.algorithm);
                                            return (
                                                <tr
                                                    key={upstream.id}
                                                    style={{
                                                        borderTop: idx > 0 ? '1px solid #f1f5f9' : 'none',
                                                        backgroundColor: idx % 2 === 0 ? '#ffffff' : '#fafafa',
                                                        transition: 'background 100ms ease'
                                                    }}
                                                >
                                                    <td style={{ padding: '12px 20px' }}>
                                                        <div style={{ display: 'flex', flexDirection: 'column' }}>
                                                            <span style={{ fontWeight: 600, color: '#0f172a', fontSize: '14px' }}>
                                                                {upstream.name}
                                                            </span>
                                                            <span style={{ fontSize: '11px', color: '#94a3b8', marginTop: '2px' }}>
                                                                Slots: {upstream.slots || 10000}
                                                            </span>
                                                        </div>
                                                    </td>

                                                    <td style={{ padding: '12px 20px' }}>
                                                        <span style={{
                                                            fontSize: '11.5px',
                                                            fontWeight: 600,
                                                            padding: '3px 8px',
                                                            borderRadius: '6px',
                                                            backgroundColor: badge.bg,
                                                            color: badge.color
                                                        }}>
                                                            {badge.label}
                                                        </span>
                                                    </td>

                                                    <td style={{ padding: '12px 20px' }}>
                                                        <button
                                                            onClick={() => handleOpenTargetsDrawer(upstream)}
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
                                                            <IconServer width={13} height={13} style={{ color: '#2563eb' }} />
                                                            <span>{upstream.targets_count || 0} Targets ({upstream.healthy_count || 0} Healthy)</span>
                                                        </button>
                                                    </td>

                                                    <td style={{ padding: '12px 20px', color: '#475569', fontSize: '12.5px' }}>
                                                        {upstream.health_check_enabled ? (
                                                            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                                <span style={{ width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#16a34a' }}></span>
                                                                <span>HTTP {upstream.health_check_path || '/health'} ({upstream.health_check_interval || 10}s)</span>
                                                            </div>
                                                        ) : (
                                                            <span style={{ color: '#94a3b8' }}>Disabled</span>
                                                        )}
                                                    </td>

                                                    <td style={{ padding: '12px 20px' }}>
                                                        <span style={{
                                                            fontSize: '11px',
                                                            fontWeight: 600,
                                                            padding: '2px 8px',
                                                            borderRadius: '12px',
                                                            backgroundColor: upstream.enabled ? '#dcfce7' : '#fee2e2',
                                                            color: upstream.enabled ? '#16a34a' : '#dc2626'
                                                        }}>
                                                            {upstream.enabled ? 'Active' : 'Disabled'}
                                                        </span>
                                                    </td>

                                                    <td style={{ padding: '12px 20px', textAlign: 'right' }}>
                                                        <div style={{ display: 'inline-flex', gap: '8px' }}>
                                                            {upstream.health_check_enabled && (
                                                                <button
                                                                    onClick={() => handleCheckPoolHealth(upstream)}
                                                                    title="Run Active Health Check Now"
                                                                    disabled={isCheckingAllHealth}
                                                                    style={{
                                                                        border: '1px solid #e2e8f0',
                                                                        backgroundColor: '#ffffff',
                                                                        padding: '6px',
                                                                        borderRadius: '6px',
                                                                        color: '#16a34a',
                                                                        cursor: 'pointer'
                                                                    }}
                                                                >
                                                                    <IconRefresh width={14} height={14} />
                                                                </button>
                                                            )}
                                                            <button
                                                                onClick={() => handleOpenTargetsDrawer(upstream)}
                                                                title="Manage Targets"
                                                                style={{
                                                                    border: '1px solid #e2e8f0',
                                                                    backgroundColor: '#ffffff',
                                                                    padding: '6px',
                                                                    borderRadius: '6px',
                                                                    color: '#2563eb',
                                                                    cursor: 'pointer'
                                                                }}
                                                            >
                                                                <IconServer width={14} height={14} />
                                                            </button>
                                                            {canManage && (
                                                                <button
                                                                    onClick={() => handleOpenEditUpstream(upstream)}
                                                                    title="Edit Upstream"
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
                                                            )}
                                                            {canManage && (
                                                                <button
                                                                    onClick={() => handleDeleteUpstream(upstream)}
                                                                    title="Delete Upstream"
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

            {/* DRAWER / MODAL: TARGETS MANAGEMENT */}
            {/* ========================================================= */}
            {selectedUpstreamForTargets && (
                <Modal
                    isOpen={!!selectedUpstreamForTargets}
                    onClose={() => setSelectedUpstreamForTargets(null)}
                    title={`Targets for: ${selectedUpstreamForTargets.name}`}
                >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                            <p style={{ fontSize: '13px', color: '#64748b', margin: 0 }}>
                                Backend server instances load-balanced using <strong>{selectedUpstreamForTargets.algorithm}</strong>.
                            </p>
                            {canManage && !isAddTargetOpen && (
                                <button
                                    className="btn-header-new-route"
                                    onClick={() => setIsAddTargetOpen(true)}
                                    style={{ fontSize: '12.5px', padding: '6px 12px' }}
                                >
                                    <IconPlus width={13} height={13} />
                                    <span>Add Target</span>
                                </button>
                            )}
                        </div>

                        {/* Inline Form to Add Target */}
                        {isAddTargetOpen && (
                            <form
                                onSubmit={handleAddTarget}
                                style={{
                                    border: '1px solid #bfdbfe',
                                    borderRadius: '8px',
                                    padding: '14px',
                                    backgroundColor: '#eff6ff',
                                }}
                            >
                                <span style={{ fontSize: '13px', fontWeight: 600, color: '#1e40af', display: 'block', marginBottom: '10px' }}>
                                    Add Backend Instance Target
                                </span>
                                <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '10px', marginBottom: '10px' }}>
                                    <div>
                                        <label className="label" style={{ fontSize: '12px' }}>Target (host:port) *</label>
                                        <input
                                            type="text"
                                            className="input"
                                            placeholder="10.10.10.11:8080 or backend.internal:3000"
                                            required
                                            value={targetForm.target}
                                            onChange={e => setTargetForm({ ...targetForm, target: e.target.value })}
                                        />
                                    </div>
                                    <div>
                                        <label className="label" style={{ fontSize: '12px' }}>Weight (0-1000)</label>
                                        <input
                                            type="number"
                                            className="input"
                                            min={0}
                                            max={1000}
                                            value={targetForm.weight}
                                            onChange={e => setTargetForm({ ...targetForm, weight: parseInt(e.target.value) || 100 })}
                                        />
                                    </div>
                                </div>
                                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                                    <button
                                        type="button"
                                        className="btn-header-refresh"
                                        onClick={() => setIsAddTargetOpen(false)}
                                        style={{ fontSize: '12px', padding: '6px 10px' }}
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        type="submit"
                                        className="btn-header-new-route"
                                        style={{ fontSize: '12px', padding: '6px 12px' }}
                                    >
                                        Add Target
                                    </button>
                                </div>
                            </form>
                        )}

                        {/* Targets List */}
                        {isTargetsLoading ? (
                            <div style={{ padding: '30px', textAlign: 'center' }}>
                                <div className="spinner" style={{ margin: '0 auto 8px auto' }}></div>
                                <span style={{ fontSize: '13px', color: '#64748b' }}>Loading targets...</span>
                            </div>
                        ) : targets.length === 0 ? (
                            <div style={{
                                padding: '32px 16px',
                                textAlign: 'center',
                                backgroundColor: '#f8fafc',
                                borderRadius: '8px',
                                border: '1px dashed #cbd5e1'
                            }}>
                                <IconServer width={28} height={28} style={{ color: '#94a3b8', margin: '0 auto 8px auto' }} />
                                <div style={{ fontSize: '14px', fontWeight: 600, color: '#334155' }}>No Targets in Pool</div>
                                <p style={{ fontSize: '12.5px', color: '#64748b', margin: '4px 0 12px 0' }}>
                                    Add at least one backend instance (e.g. 10.0.0.1:8080) to route traffic.
                                </p>
                                {canManage && !isAddTargetOpen && (
                                    <button
                                        className="btn-header-new-route"
                                        onClick={() => setIsAddTargetOpen(true)}
                                    >
                                        Add First Target
                                    </button>
                                )}
                            </div>
                        ) : (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                {targets.map(target => (
                                    <div
                                        key={target.id}
                                        style={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            justifyContent: 'space-between',
                                            padding: '12px 14px',
                                            backgroundColor: '#ffffff',
                                            border: '1px solid #e2e8f0',
                                            borderRadius: '8px',
                                        }}
                                    >
                                        <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                                <code style={{
                                                    fontFamily: 'monospace',
                                                    fontSize: '13px',
                                                    fontWeight: 600,
                                                    color: '#0f172a'
                                                }}>
                                                    {target.target}
                                                </code>
                                                <span style={{
                                                    fontSize: '11px',
                                                    padding: '1px 6px',
                                                    borderRadius: '4px',
                                                    backgroundColor:
                                                        target.health_status === 'healthy' ? '#dcfce7' :
                                                        target.health_status === 'unhealthy' ? '#fee2e2' : '#f1f5f9',
                                                    color:
                                                        target.health_status === 'healthy' ? '#16a34a' :
                                                        target.health_status === 'unhealthy' ? '#dc2626' : '#64748b',
                                                    fontWeight: 600,
                                                    textTransform: 'capitalize'
                                                }}>
                                                    {target.health_status}
                                                </span>
                                            </div>
                                            <span style={{ fontSize: '11.5px', color: '#64748b' }}>
                                                Weight: <strong>{target.weight}</strong> | {target.enabled ? 'Active' : 'Disabled'}
                                            </span>
                                        </div>

                                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                            <button
                                                onClick={() => handleCheckSingleTarget(target)}
                                                disabled={healthCheckingTargetId === target.id}
                                                title="Check Health"
                                                style={{
                                                    border: '1px solid #e2e8f0',
                                                    backgroundColor: '#ffffff',
                                                    padding: '5px 10px',
                                                    borderRadius: '6px',
                                                    fontSize: '12px',
                                                    cursor: 'pointer',
                                                    color: '#16a34a'
                                                }}
                                            >
                                                {healthCheckingTargetId === target.id ? 'Checking...' : 'Check Health'}
                                            </button>
                                            {canManage && (
                                                <button
                                                    onClick={() => handleToggleTarget(target)}
                                                    style={{
                                                        border: '1px solid #e2e8f0',
                                                        backgroundColor: '#ffffff',
                                                        padding: '5px 10px',
                                                        borderRadius: '6px',
                                                        fontSize: '12px',
                                                        cursor: 'pointer',
                                                        color: target.enabled ? '#d97706' : '#16a34a'
                                                    }}
                                                >
                                                    {target.enabled ? 'Disable' : 'Enable'}
                                                </button>
                                            )}
                                            {canManage && (
                                                <button
                                                    onClick={() => handleDeleteTarget(target.id)}
                                                    title="Remove Target"
                                                    style={{
                                                        border: '1px solid #fee2e2',
                                                        backgroundColor: '#ffffff',
                                                        padding: '5px 8px',
                                                        borderRadius: '6px',
                                                        fontSize: '12px',
                                                        cursor: 'pointer',
                                                        color: '#dc2626'
                                                    }}
                                                >
                                                    <IconTrash width={13} height={13} />
                                                </button>
                                            )}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}

                        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '12px' }}>
                            <button className="btn-header-refresh" onClick={() => setSelectedUpstreamForTargets(null)}>
                                Close
                            </button>
                        </div>
                    </div>
                </Modal>
            )}
        </div>
    );
}
