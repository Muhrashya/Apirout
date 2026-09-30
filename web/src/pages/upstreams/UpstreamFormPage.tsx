import { useState, useEffect } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import { gatewayApi, Upstream, Target } from '../../api/gatewayApi';
import { useToast } from '../../components/Toast';
import { IconArrowLeft, IconPlus, IconTrash } from '../../components/Icons';

export default function UpstreamFormPage() {
    const { id } = useParams<{ id: string }>();
    const isEdit = Boolean(id);
    const navigate = useNavigate();
    const toast = useToast();

    const [isLoading, setIsLoading] = useState(isEdit);
    const [isSaving, setIsSaving] = useState(false);

    const [formData, setFormData] = useState({
        name: '',
        algorithm: 'round-robin' as 'round-robin' | 'least-connections' | 'consistent-hashing',
        hash_on: 'none',
        hash_fallback: 'none',
        slots: 10000,
        health_check_path: '/healthz',
        health_check_interval: 10,
        health_check_timeout: 3,
        healthy_threshold: 2,
        unhealthy_threshold: 3
    });

    const [targets, setTargets] = useState<{ host: string; port: number; weight: number; enabled: boolean }[]>([]);
    const [newTargetHost, setNewTargetHost] = useState('');
    const [newTargetPort, setNewTargetPort] = useState(8080);
    const [newTargetWeight, setNewTargetWeight] = useState(100);

    useEffect(() => {
        if (isEdit && id) {
            loadUpstreamData();
        }
    }, [id]);

    const loadUpstreamData = async () => {
        try {
            const [upstreamRes, targetsRes] = await Promise.all([
                gatewayApi.getUpstream(id!),
                gatewayApi.getTargets(id!).catch(() => ({ data: [] }))
            ]);

            const u: any = (upstreamRes as any)?.data || upstreamRes;
            if (u) {
                setFormData({
                    name: u.name,
                    algorithm: u.algorithm || 'round-robin',
                    hash_on: u.hash_on || 'none',
                    hash_fallback: u.hash_fallback || 'none',
                    slots: u.slots || 10000,
                    health_check_path: u.health_check_path || '/healthz',
                    health_check_interval: u.health_check_interval || 10,
                    health_check_timeout: u.health_check_timeout || 3,
                    healthy_threshold: u.healthy_threshold || 2,
                    unhealthy_threshold: u.unhealthy_threshold || 3
                });
            }

            if (targetsRes.data) {
                setTargets(targetsRes.data.map((t: Target) => ({
                    host: t.host,
                    port: t.port,
                    weight: t.weight,
                    enabled: t.enabled
                })));
            }
        } catch (err: any) {
            toast.error(err.message || 'Failed to load upstream data');
        } finally {
            setIsLoading(false);
        }
    };

    const handleAddTarget = () => {
        if (!newTargetHost.trim()) {
            toast.error('Target Host is required');
            return;
        }

        setTargets(prev => [
            ...prev,
            {
                host: newTargetHost.trim(),
                port: newTargetPort,
                weight: newTargetWeight,
                enabled: true
            }
        ]);

        setNewTargetHost('');
        setNewTargetPort(8080);
        setNewTargetWeight(100);
    };

    const handleRemoveTarget = (index: number) => {
        setTargets(prev => prev.filter((_, i) => i !== index));
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!formData.name.trim()) {
            toast.error('Upstream Name is required');
            return;
        }

        const payload: Partial<Upstream> = {
            name: formData.name.trim(),
            algorithm: formData.algorithm,
            hash_on: formData.hash_on,
            hash_fallback: formData.hash_fallback,
            slots: formData.slots,
            health_check_path: formData.health_check_path,
            health_check_interval: formData.health_check_interval,
            health_check_timeout: formData.health_check_timeout,
            healthy_threshold: formData.healthy_threshold,
            unhealthy_threshold: formData.unhealthy_threshold
        };

        setIsSaving(true);
        try {
            let upstreamId = id;
            if (isEdit && id) {
                await gatewayApi.updateUpstream(id, payload);
                toast.success('Upstream pool updated');
            } else {
                const res = await gatewayApi.createUpstream(payload);
                upstreamId = (res as any)?.data?.id || (res as any)?.id;
                toast.success('Upstream pool created');
            }

            // Sync targets if newly created
            if (!isEdit && upstreamId && targets.length > 0) {
                for (const t of targets) {
                    await gatewayApi.createTarget(upstreamId, {
                        target: `${t.host}:${t.port}`,
                        host: t.host,
                        port: t.port,
                        weight: t.weight,
                        enabled: t.enabled
                    }).catch(() => null);
                }
            }

            navigate('/upstreams');
        } catch (err: any) {
            toast.error(err.message || 'Failed to save upstream');
        } finally {
            setIsSaving(false);
        }
    };

    if (isLoading) {
        return (
            <div style={{ padding: '60px', textAlign: 'center' }}>
                <div className="spinner" style={{ margin: '0 auto' }}></div>
                <div style={{ marginTop: '16px', color: '#64748b' }}>Loading upstream...</div>
            </div>
        );
    }

    return (
        <div className="overview-page-wrapper" style={{ paddingBottom: '60px' }}>
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Gateway</span>
                <span className="breadcrumb-sep">/</span>
                <Link to="/upstreams" style={{ color: '#2563eb', textDecoration: 'none' }}>
                    Upstreams
                </Link>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">{isEdit ? 'Edit Upstream' : 'New Upstream'}</span>
            </div>

            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">
                        {isEdit ? 'Edit Upstream Pool' : 'New Upstream Pool'}
                    </h1>
                    <p className="overview-subtitle">
                        Configure dynamic backend target pools, load-balancing algorithms, and active health checks.
                    </p>
                </div>
                <div className="overview-header-actions">
                    <Link to="/upstreams" className="btn-header-refresh" style={{ textDecoration: 'none' }}>
                        <IconArrowLeft width={14} height={14} />
                        <span>Back to Upstreams</span>
                    </Link>
                </div>
            </div>

            <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '20px', width: '100%' }}>
                {/* Card 1: Algorithm & Pool Settings */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Pool Configuration &amp; Algorithm
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                        <div>
                            <label className="label">Upstream Name *</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="e.g. payment-service.pool"
                                required
                                value={formData.name}
                                onChange={e => setFormData(prev => ({ ...prev, name: e.target.value }))}
                            />
                        </div>

                        <div>
                            <label className="label">Load Balancing Algorithm</label>
                            <select
                                className="select"
                                value={formData.algorithm}
                                onChange={e => setFormData(prev => ({ ...prev, algorithm: e.target.value as any }))}
                            >
                                <option value="round-robin">Round Robin (default)</option>
                                <option value="least-connections">Least Connections</option>
                                <option value="consistent-hashing">Consistent Hashing</option>
                            </select>
                        </div>
                    </div>

                    {formData.algorithm === 'consistent-hashing' && (
                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', padding: '14px', background: '#f8fafc', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '16px' }}>
                            <div>
                                <label className="label">Hash On</label>
                                <select
                                    className="select"
                                    value={formData.hash_on}
                                    onChange={e => setFormData(prev => ({ ...prev, hash_on: e.target.value }))}
                                >
                                    <option value="none">none</option>
                                    <option value="ip">Client IP Address</option>
                                    <option value="header">HTTP Header</option>
                                    <option value="cookie">Cookie</option>
                                </select>
                            </div>
                            <div>
                                <label className="label">Hash Fallback</label>
                                <select
                                    className="select"
                                    value={formData.hash_fallback}
                                    onChange={e => setFormData(prev => ({ ...prev, hash_fallback: e.target.value }))}
                                >
                                    <option value="none">none</option>
                                    <option value="ip">Client IP</option>
                                    <option value="header">HTTP Header</option>
                                </select>
                            </div>
                        </div>
                    )}
                </div>

                {/* Card 2: Active Health Checks */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Active Health Checks (Passive &amp; Active Probes)
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr', gap: '14px', marginBottom: '14px' }}>
                        <div>
                            <label className="label">Health Check Path</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="/healthz"
                                value={formData.health_check_path}
                                onChange={e => setFormData(prev => ({ ...prev, health_check_path: e.target.value }))}
                            />
                        </div>
                        <div>
                            <label className="label">Probe Interval (s)</label>
                            <input
                                type="number"
                                className="input"
                                min={1}
                                value={formData.health_check_interval}
                                onChange={e => setFormData(prev => ({ ...prev, health_check_interval: parseInt(e.target.value) || 10 }))}
                            />
                        </div>
                        <div>
                            <label className="label">Probe Timeout (s)</label>
                            <input
                                type="number"
                                className="input"
                                min={1}
                                value={formData.health_check_timeout}
                                onChange={e => setFormData(prev => ({ ...prev, health_check_timeout: parseInt(e.target.value) || 3 }))}
                            />
                        </div>
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
                        <div>
                            <label className="label">Healthy Threshold (consecutive successes)</label>
                            <input
                                type="number"
                                className="input"
                                min={1}
                                value={formData.healthy_threshold}
                                onChange={e => setFormData(prev => ({ ...prev, healthy_threshold: parseInt(e.target.value) || 2 }))}
                            />
                        </div>
                        <div>
                            <label className="label">Unhealthy Threshold (consecutive failures)</label>
                            <input
                                type="number"
                                className="input"
                                min={1}
                                value={formData.unhealthy_threshold}
                                onChange={e => setFormData(prev => ({ ...prev, unhealthy_threshold: parseInt(e.target.value) || 3 }))}
                            />
                        </div>
                    </div>
                </div>

                {/* Card 3: Inline Target Instances */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Target Instances (Endpoints in Pool)
                    </h2>

                    {/* Add target form */}
                    <div style={{ display: 'grid', gridTemplateColumns: '2fr 100px 100px auto', gap: '10px', alignItems: 'end', marginBottom: '16px', background: '#f8fafc', padding: '14px', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                        <div>
                            <label className="label">Target Host/IP</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="10.0.0.12 or node1.internal"
                                value={newTargetHost}
                                onChange={e => setNewTargetHost(e.target.value)}
                            />
                        </div>
                        <div>
                            <label className="label">Port</label>
                            <input
                                type="number"
                                className="input"
                                min={1}
                                max={65535}
                                value={newTargetPort}
                                onChange={e => setNewTargetPort(parseInt(e.target.value) || 80)}
                            />
                        </div>
                        <div>
                            <label className="label">Weight</label>
                            <input
                                type="number"
                                className="input"
                                min={1}
                                max={1000}
                                value={newTargetWeight}
                                onChange={e => setNewTargetWeight(parseInt(e.target.value) || 100)}
                            />
                        </div>
                        <button
                            type="button"
                            className="btn-header-new-route"
                            onClick={handleAddTarget}
                            style={{ height: '38px', padding: '0 14px' }}
                        >
                            <IconPlus width={14} height={14} />
                            <span>Add</span>
                        </button>
                    </div>

                    {/* Target list */}
                    {targets.length > 0 ? (
                        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                            <thead>
                                <tr style={{ borderBottom: '1px solid #e2e8f0', color: '#64748b', textAlign: 'left' }}>
                                    <th style={{ padding: '8px 12px' }}>Target Destination</th>
                                    <th style={{ padding: '8px 12px' }}>Weight</th>
                                    <th style={{ padding: '8px 12px' }}>Status</th>
                                    <th style={{ padding: '8px 12px', textAlign: 'right' }}>Action</th>
                                </tr>
                            </thead>
                            <tbody>
                                {targets.map((t, idx) => (
                                    <tr key={idx} style={{ borderBottom: '1px solid #f1f5f9' }}>
                                        <td style={{ padding: '10px 12px', fontWeight: 600, color: '#0f172a' }}>
                                            {t.host}:{t.port}
                                        </td>
                                        <td style={{ padding: '10px 12px', color: '#475569' }}>
                                            {t.weight}
                                        </td>
                                        <td style={{ padding: '10px 12px' }}>
                                            <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '12px', background: '#dcfce7', color: '#16a34a', fontWeight: 600 }}>
                                                Enabled
                                            </span>
                                        </td>
                                        <td style={{ padding: '10px 12px', textAlign: 'right' }}>
                                            <button
                                                type="button"
                                                onClick={() => handleRemoveTarget(idx)}
                                                style={{ border: 'none', background: 'transparent', color: '#ef4444', cursor: 'pointer' }}
                                                title="Remove Target"
                                            >
                                                <IconTrash width={15} height={15} />
                                            </button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    ) : (
                        <div style={{ textAlign: 'center', padding: '20px', color: '#94a3b8', fontSize: '13px' }}>
                            No target instances added yet. You can add them above or after creating the upstream pool.
                        </div>
                    )}
                </div>

                {/* Bottom Action Bar */}
                <div style={{
                    display: 'flex',
                    justifyContent: 'flex-end',
                    gap: '12px',
                    padding: '16px 24px',
                    background: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    boxShadow: '0 2px 8px rgba(0,0,0,0.06)'
                }}>
                    <button
                        type="button"
                        className="btn-header-refresh"
                        onClick={() => navigate('/upstreams')}
                        disabled={isSaving}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="btn-header-new-route"
                        disabled={isSaving}
                    >
                        {isSaving ? 'Saving Upstream...' : isEdit ? 'Update Upstream' : 'Create Upstream Pool'}
                    </button>
                </div>
            </form>
        </div>
    );
}
