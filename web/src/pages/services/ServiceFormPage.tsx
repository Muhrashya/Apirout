import { useState, useEffect } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import { gatewayApi, GatewayService, Upstream, Connection } from '../../api/gatewayApi';
import { useToast } from '../../components/Toast';
import {
    IconServer, IconLayers, IconDatabase, IconArrowLeft
} from '../../components/Icons';
import { AuthPluginManager } from '../../components/AuthPluginManager';

export default function ServiceFormPage() {
    const { id } = useParams<{ id: string }>();
    const isEdit = Boolean(id);
    const navigate = useNavigate();
    const toast = useToast();

    const [isLoading, setIsLoading] = useState(isEdit);
    const [isSaving, setIsSaving] = useState(false);
    const [upstreams, setUpstreams] = useState<Upstream[]>([]);
    const [connections, setConnections] = useState<Connection[]>([]);

    const [targetMode, setTargetMode] = useState<'direct' | 'upstream' | 'database'>('direct');
    const [formData, setFormData] = useState({
        name: '',
        description: '',
        protocol: 'http' as 'http' | 'https' | 'grpc' | 'tcp',
        host: '',
        port: 80,
        path: '',
        connect_timeout: 60000,
        read_timeout: 60000,
        write_timeout: 60000,
        retries: 5,
        enabled: true,
        upstream_id: '',
        connection_id: ''
    });

    const [outboundAuth, setOutboundAuth] = useState<{
        enabled: boolean;
        auth_type: string;
        config: Record<string, any>;
    }>({
        enabled: false,
        auth_type: '',
        config: {}
    });
    const [showOutboundSecret, setShowOutboundSecret] = useState(false);

    useEffect(() => {
        loadPrerequisites();
    }, [id]);

    const loadPrerequisites = async () => {
        try {
            const [upstreamsRes, connsRes] = await Promise.all([
                gatewayApi.getUpstreams().catch(() => ({ data: [] })),
                gatewayApi.getConnections().catch(() => ({ data: [] }))
            ]);

            setUpstreams(upstreamsRes.data || []);
            setConnections(connsRes.data || []);

            if (isEdit && id) {
                const serviceRes = await gatewayApi.getService(id);
                const s: any = (serviceRes as any)?.data || serviceRes;
                if (s) {
                    if (s.connection_id) {
                        setTargetMode('database');
                    } else if (s.upstream_id) {
                        setTargetMode('upstream');
                    } else {
                        setTargetMode('direct');
                    }

                    setFormData({
                        name: s.name,
                        description: s.description || '',
                        protocol: s.protocol || 'http',
                        host: s.host || '',
                        port: s.port || 80,
                        path: s.path || '',
                        connect_timeout: s.connect_timeout || 60000,
                        read_timeout: s.read_timeout || 60000,
                        write_timeout: s.write_timeout || 60000,
                        retries: s.retries ?? 5,
                        enabled: s.enabled ?? true,
                        upstream_id: s.upstream_id || '',
                        connection_id: s.connection_id || ''
                    });

                    // Load Outbound Auth config
                    try {
                        const outRes = await gatewayApi.getOutboundAuth(id);
                        const outData = (outRes as any)?.data || outRes;
                        if (outData && outData.auth_type) {
                            let parsedCfg = {};
                            try {
                                parsedCfg = typeof outData.config === 'string' ? JSON.parse(outData.config) : outData.config;
                            } catch {}
                            setOutboundAuth({
                                enabled: outData.enabled ?? true,
                                auth_type: outData.auth_type,
                                config: parsedCfg
                            });
                        }
                    } catch (e) {
                        console.error('Failed to load outbound auth', e);
                    }
                }
            }
        } catch (err: any) {
            toast.error(err.message || 'Failed to load service data');
        } finally {
            setIsLoading(false);
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!formData.name.trim()) {
            toast.error('Service Name is required');
            return;
        }

        const payload: Partial<GatewayService> = {
            name: formData.name.trim(),
            description: formData.description.trim(),
            path: formData.path.trim(),
            connect_timeout: formData.connect_timeout,
            read_timeout: formData.read_timeout,
            write_timeout: formData.write_timeout,
            retries: formData.retries,
            enabled: formData.enabled
        };

        if (targetMode === 'direct') {
            if (!formData.host.trim()) {
                toast.error('Host is required for Direct Backend');
                return;
            }
            payload.protocol = formData.protocol;
            payload.host = formData.host.trim();
            payload.port = formData.port;
            payload.upstream_id = undefined;
            payload.connection_id = undefined;
        } else if (targetMode === 'upstream') {
            if (!formData.upstream_id) {
                toast.error('Please select an Upstream pool');
                return;
            }
            payload.upstream_id = formData.upstream_id;
            payload.connection_id = undefined;
            payload.host = '';
            payload.port = 80;
        } else if (targetMode === 'database') {
            if (!formData.connection_id) {
                toast.error('Please select a Database Connection profile');
                return;
            }
            payload.connection_id = formData.connection_id;
            payload.upstream_id = undefined;
            payload.protocol = 'http';
            payload.host = '';
            payload.port = 80;
        }

        setIsSaving(true);
        try {
            let serviceId = id;
            if (isEdit && id) {
                await gatewayApi.updateService(id, payload);
            } else {
                const created: any = await gatewayApi.createService(payload);
                serviceId = created?.data?.id || created?.id;
            }

            if (serviceId) {
                if (outboundAuth.enabled && outboundAuth.auth_type) {
                    await gatewayApi.setOutboundAuth(serviceId, {
                        auth_type: outboundAuth.auth_type,
                        config: outboundAuth.config as any,
                        enabled: true
                    });
                } else if (isEdit) {
                    await gatewayApi.deleteOutboundAuth(serviceId).catch(() => {});
                }
            }

            toast.success(isEdit ? 'Gateway Service updated successfully' : 'Gateway Service created successfully');
            navigate('/services');
        } catch (err: any) {
            toast.error(err.message || 'Failed to save service');
        } finally {
            setIsSaving(false);
        }
    };

    if (isLoading) {
        return (
            <div style={{ padding: '60px', textAlign: 'center' }}>
                <div className="spinner" style={{ margin: '0 auto' }}></div>
                <div style={{ marginTop: '16px', color: '#64748b' }}>Loading service...</div>
            </div>
        );
    }

    return (
        <div className="overview-page-wrapper" style={{ paddingBottom: '60px' }}>
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Gateway</span>
                <span className="breadcrumb-sep">/</span>
                <Link to="/services" style={{ color: '#2563eb', textDecoration: 'none' }}>
                    Services
                </Link>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">{isEdit ? 'Edit Service' : 'New Service'}</span>
            </div>

            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">
                        {isEdit ? `Edit Gateway Service` : 'New Gateway Service'}
                    </h1>
                    <p className="overview-subtitle">
                        Configure logical service endpoints, upstream load-balancing pools, or database Instant API sources.
                    </p>
                </div>
                <div className="overview-header-actions">
                    <Link to="/services" className="btn-header-refresh" style={{ textDecoration: 'none' }}>
                        <IconArrowLeft width={14} height={14} />
                        <span>Back to Services</span>
                    </Link>
                </div>
            </div>

            <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '20px', width: '100%' }}>
                {/* Card 1: General Info */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Service Information
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '16px' }}>
                        <div className="form-group">
                            <label className="label">Service Name *</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="e.g. user-service or payments-api"
                                required
                                value={formData.name}
                                onChange={e => setFormData(prev => ({ ...prev, name: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Logical name used by routes and downstream consumers.
                            </span>
                        </div>

                        <div className="form-group">
                            <label className="label">Description</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="Brief description of this service"
                                value={formData.description}
                                onChange={e => setFormData(prev => ({ ...prev, description: e.target.value }))}
                            />
                        </div>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <input
                            type="checkbox"
                            id="svc-enabled"
                            checked={formData.enabled}
                            onChange={e => setFormData(prev => ({ ...prev, enabled: e.target.checked }))}
                            style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                        />
                        <label htmlFor="svc-enabled" style={{ fontSize: '13.5px', fontWeight: 500, color: '#1e293b', cursor: 'pointer' }}>
                            Enable this Service (routes linked to this service will accept traffic)
                        </label>
                    </div>
                </div>

                {/* Card 2: Target Architecture */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Target Architecture &amp; Destination
                    </h2>

                    <label className="label" style={{ marginBottom: '10px' }}>Select Backend Target Type *</label>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '14px', marginBottom: '20px' }}>
                        <div
                            onClick={() => setTargetMode('direct')}
                            style={{
                                border: targetMode === 'direct' ? '2px solid #2563eb' : '2px solid #e2e8f0',
                                background: targetMode === 'direct' ? '#eff6ff' : '#ffffff',
                                borderRadius: '10px',
                                padding: '16px',
                                cursor: 'pointer',
                                textAlign: 'center',
                                transition: 'border-color 150ms ease, background-color 150ms ease'
                            }}
                        >
                            <IconServer width={24} height={24} style={{ margin: '0 auto 6px', color: targetMode === 'direct' ? '#2563eb' : '#64748b' }} />
                            <div style={{ fontWeight: 600, fontSize: '14px', color: targetMode === 'direct' ? '#1d4ed8' : '#1e293b' }}>Direct Backend</div>
                            <div style={{ fontSize: '12px', color: '#64748b', marginTop: '2px' }}>Single host &amp; port forwarding</div>
                        </div>

                        <div
                            onClick={() => setTargetMode('upstream')}
                            style={{
                                border: targetMode === 'upstream' ? '2px solid #2563eb' : '2px solid #e2e8f0',
                                background: targetMode === 'upstream' ? '#eff6ff' : '#ffffff',
                                borderRadius: '10px',
                                padding: '16px',
                                cursor: 'pointer',
                                textAlign: 'center',
                                transition: 'border-color 150ms ease, background-color 150ms ease'
                            }}
                        >
                            <IconLayers width={24} height={24} style={{ margin: '0 auto 6px', color: targetMode === 'upstream' ? '#2563eb' : '#64748b' }} />
                            <div style={{ fontWeight: 600, fontSize: '14px', color: targetMode === 'upstream' ? '#1d4ed8' : '#1e293b' }}>Upstream Pool</div>
                            <div style={{ fontSize: '12px', color: '#64748b', marginTop: '2px' }}>Load-balanced target cluster</div>
                        </div>

                        <div
                            onClick={() => setTargetMode('database')}
                            style={{
                                border: targetMode === 'database' ? '2px solid #2563eb' : '2px solid #e2e8f0',
                                background: targetMode === 'database' ? '#eff6ff' : '#ffffff',
                                borderRadius: '10px',
                                padding: '16px',
                                cursor: 'pointer',
                                textAlign: 'center',
                                transition: 'border-color 150ms ease, background-color 150ms ease'
                            }}
                        >
                            <IconDatabase width={24} height={24} style={{ margin: '0 auto 6px', color: targetMode === 'database' ? '#2563eb' : '#64748b' }} />
                            <div style={{ fontWeight: 600, fontSize: '14px', color: targetMode === 'database' ? '#1d4ed8' : '#1e293b' }}>Data Source</div>
                            <div style={{ fontSize: '12px', color: '#64748b', marginTop: '2px' }}>Instant CRUD &amp; SQL execution</div>
                        </div>
                    </div>

                    {/* Mode Direct */}
                    {targetMode === 'direct' && (
                        <div style={{ background: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '8px', padding: '18px' }}>
                            <div style={{ display: 'grid', gridTemplateColumns: '140px 1fr 120px', gap: '12px', marginBottom: '14px' }}>
                                <div>
                                    <label className="label">Protocol</label>
                                    <select
                                        className="select"
                                        value={formData.protocol}
                                        onChange={e => setFormData(prev => ({ ...prev, protocol: e.target.value as any }))}
                                    >
                                        <option value="http">http</option>
                                        <option value="https">https</option>
                                        <option value="grpc">grpc</option>
                                        <option value="tcp">tcp</option>
                                    </select>
                                </div>
                                <div>
                                    <label className="label">Host *</label>
                                    <input
                                        type="text"
                                        className="input"
                                        placeholder="e.g. backend-api.internal or 192.168.1.10"
                                        required
                                        value={formData.host}
                                        onChange={e => setFormData(prev => ({ ...prev, host: e.target.value }))}
                                    />
                                </div>
                                <div>
                                    <label className="label">Port *</label>
                                    <input
                                        type="number"
                                        className="input"
                                        min={1}
                                        max={65535}
                                        value={formData.port}
                                        onChange={e => setFormData(prev => ({ ...prev, port: parseInt(e.target.value) || 80 }))}
                                    />
                                </div>
                            </div>
                            <div>
                                <label className="label">Base Path (Optional)</label>
                                <input
                                    type="text"
                                    className="input"
                                    placeholder="e.g. /v1 or /api"
                                    value={formData.path}
                                    onChange={e => setFormData(prev => ({ ...prev, path: e.target.value }))}
                                />
                                <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                    Prepended to upstream requests forwarded to this destination.
                                </span>
                            </div>
                        </div>
                    )}

                    {/* Mode Upstream */}
                    {targetMode === 'upstream' && (
                        <div style={{ background: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '8px', padding: '18px' }}>
                            <div className="form-group" style={{ marginBottom: '14px' }}>
                                <label className="label">Select Upstream Load Balancing Pool *</label>
                                <select
                                    className="select"
                                    required
                                    value={formData.upstream_id}
                                    onChange={e => setFormData(prev => ({ ...prev, upstream_id: e.target.value }))}
                                >
                                    <option value="">-- Choose Upstream Pool --</option>
                                    {upstreams.map(u => (
                                        <option key={u.id} value={u.id}>
                                            {u.name} ({u.algorithm} — {u.targets_count || 0} targets)
                                        </option>
                                    ))}
                                </select>
                            </div>
                            <div>
                                <label className="label">Base Path (Optional)</label>
                                <input
                                    type="text"
                                    className="input"
                                    placeholder="e.g. /v1"
                                    value={formData.path}
                                    onChange={e => setFormData(prev => ({ ...prev, path: e.target.value }))}
                                />
                            </div>
                        </div>
                    )}

                    {/* Mode Database */}
                    {targetMode === 'database' && (
                        <div style={{ background: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '8px', padding: '18px' }}>
                            <div className="form-group">
                                <label className="label">Select Database Connection Profile *</label>
                                <select
                                    className="select"
                                    required
                                    value={formData.connection_id}
                                    onChange={e => setFormData(prev => ({ ...prev, connection_id: e.target.value }))}
                                >
                                    <option value="">-- Choose Database Connection Profile --</option>
                                    {connections.map(c => (
                                        <option key={c.id} value={c.id}>
                                            {c.name} ({c.db_type} — {c.db_name})
                                        </option>
                                    ))}
                                </select>
                                <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '6px', display: 'block' }}>
                                    Requests routed to this service will directly execute SQL queries or Instant CRUD operations against this database engine.
                                </span>
                            </div>
                        </div>
                    )}
                </div>

                {/* Card 3: Network & Timeouts */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Network &amp; Timeout Policies
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '14px' }}>
                        <div>
                            <label className="label">Connect Timeout (ms)</label>
                            <input
                                type="number"
                                className="input"
                                min={100}
                                value={formData.connect_timeout}
                                onChange={e => setFormData(prev => ({ ...prev, connect_timeout: parseInt(e.target.value) || 60000 }))}
                            />
                        </div>
                        <div>
                            <label className="label">Read Timeout (ms)</label>
                            <input
                                type="number"
                                className="input"
                                min={100}
                                value={formData.read_timeout}
                                onChange={e => setFormData(prev => ({ ...prev, read_timeout: parseInt(e.target.value) || 60000 }))}
                            />
                        </div>
                        <div>
                            <label className="label">Write Timeout (ms)</label>
                            <input
                                type="number"
                                className="input"
                                min={100}
                                value={formData.write_timeout}
                                onChange={e => setFormData(prev => ({ ...prev, write_timeout: parseInt(e.target.value) || 60000 }))}
                            />
                        </div>
                        <div>
                            <label className="label">Retries</label>
                            <input
                                type="number"
                                className="input"
                                min={0}
                                max={10}
                                value={formData.retries}
                                onChange={e => setFormData(prev => ({ ...prev, retries: parseInt(e.target.value) || 5 }))}
                            />
                        </div>
                    </div>
                </div>

                {/* Card 4: Upstream Authentication (Outbound Credential Injection) */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px', marginBottom: '16px' }}>
                        <div>
                            <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                                Upstream Authentication (Outbound Injection)
                            </h2>
                            <p style={{ fontSize: '12px', color: '#64748b', margin: '2px 0 0 0' }}>
                                Automatically inject credentials into requests proxied to your backend services. Secrets are AES-256 encrypted at rest.
                            </p>
                        </div>
                        <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', cursor: 'pointer', fontWeight: 600 }}>
                            <input
                                type="checkbox"
                                checked={outboundAuth.enabled}
                                onChange={e => setOutboundAuth(prev => ({ ...prev, enabled: e.target.checked }))}
                                style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                            />
                            Enable Outbound Auth
                        </label>
                    </div>

                    {outboundAuth.enabled && (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                            <div>
                                <label className="label">Authentication Scheme *</label>
                                <select
                                    className="select"
                                    value={outboundAuth.auth_type}
                                    onChange={e => setOutboundAuth(prev => ({ ...prev, auth_type: e.target.value }))}
                                    required={outboundAuth.enabled}
                                >
                                    <option value="">Select Scheme...</option>
                                    <option value="bearer">Bearer Token (Authorization: Bearer &lt;token&gt;)</option>
                                    <option value="basic">HTTP Basic Auth (Authorization: Basic base64(user:pass))</option>
                                    <option value="api-key">API Key (Custom Header or Query Parameter)</option>
                                    <option value="aws-sigv4">AWS Signature V4 (SigV4 signing headers)</option>
                                    <option value="oauth2-cc">OAuth2 Client Credentials (Token grant)</option>
                                </select>
                            </div>

                            {/* Bearer Token Form */}
                            {outboundAuth.auth_type === 'bearer' && (
                                <div style={{ background: '#f8fafc', padding: '16px', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 140px', gap: '14px' }}>
                                        <div>
                                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                                                <label className="label" style={{ margin: 0 }}>Bearer Token / Secret *</label>
                                                <button
                                                    type="button"
                                                    onClick={() => setShowOutboundSecret(!showOutboundSecret)}
                                                    style={{ background: 'none', border: 'none', color: '#2563eb', fontSize: '12px', cursor: 'pointer', padding: 0 }}
                                                >
                                                    {showOutboundSecret ? 'Hide Secret' : 'Show Secret'}
                                                </button>
                                            </div>
                                            <input
                                                type={showOutboundSecret ? 'text' : 'password'}
                                                className="input"
                                                placeholder={outboundAuth.config.token_encrypted ? '•••••••• (Encrypted, enter new value to change)' : 'Paste backend secret token here'}
                                                value={outboundAuth.config.token || ''}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, token: e.target.value }
                                                }))}
                                            />
                                        </div>
                                        <div>
                                            <label className="label">Auth Prefix</label>
                                            <input
                                                type="text"
                                                className="input"
                                                value={outboundAuth.config.prefix || 'Bearer'}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, prefix: e.target.value }
                                                }))}
                                            />
                                        </div>
                                    </div>
                                </div>
                            )}

                            {/* Basic Auth Form */}
                            {outboundAuth.auth_type === 'basic' && (
                                <div style={{ background: '#f8fafc', padding: '16px', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
                                        <div>
                                            <label className="label">Username *</label>
                                            <input
                                                type="text"
                                                className="input"
                                                placeholder="e.g. admin or api_service"
                                                value={outboundAuth.config.username || ''}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, username: e.target.value }
                                                }))}
                                            />
                                        </div>
                                        <div>
                                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                                                <label className="label" style={{ margin: 0 }}>Password *</label>
                                                <button
                                                    type="button"
                                                    onClick={() => setShowOutboundSecret(!showOutboundSecret)}
                                                    style={{ background: 'none', border: 'none', color: '#2563eb', fontSize: '12px', cursor: 'pointer', padding: 0 }}
                                                >
                                                    {showOutboundSecret ? 'Hide' : 'Show'}
                                                </button>
                                            </div>
                                            <input
                                                type={showOutboundSecret ? 'text' : 'password'}
                                                className="input"
                                                placeholder={outboundAuth.config.password_encrypted ? '•••••••• (Encrypted, enter new value to change)' : 'Backend password'}
                                                value={outboundAuth.config.password || ''}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, password: e.target.value }
                                                }))}
                                            />
                                        </div>
                                    </div>
                                </div>
                            )}

                            {/* API Key Form */}
                            {outboundAuth.auth_type === 'api-key' && (
                                <div style={{ background: '#f8fafc', padding: '16px', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px', marginBottom: '12px' }}>
                                        <div>
                                            <label className="label">Header / Param Name *</label>
                                            <input
                                                type="text"
                                                className="input"
                                                placeholder="X-API-Key or api_key"
                                                value={outboundAuth.config.header_name || 'X-API-Key'}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, header_name: e.target.value }
                                                }))}
                                            />
                                        </div>
                                        <div>
                                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                                                <label className="label" style={{ margin: 0 }}>API Key Secret *</label>
                                                <button
                                                    type="button"
                                                    onClick={() => setShowOutboundSecret(!showOutboundSecret)}
                                                    style={{ background: 'none', border: 'none', color: '#2563eb', fontSize: '12px', cursor: 'pointer', padding: 0 }}
                                                >
                                                    {showOutboundSecret ? 'Hide' : 'Show'}
                                                </button>
                                            </div>
                                            <input
                                                type={showOutboundSecret ? 'text' : 'password'}
                                                className="input"
                                                placeholder={outboundAuth.config.key_encrypted ? '•••••••• (Encrypted)' : 'API key value'}
                                                value={outboundAuth.config.key || ''}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, key: e.target.value }
                                                }))}
                                            />
                                        </div>
                                    </div>
                                    <div style={{ display: 'flex', gap: '16px' }}>
                                        <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12.5px', cursor: 'pointer' }}>
                                            <input
                                                type="radio"
                                                name="key_location"
                                                checked={!outboundAuth.config.in_query}
                                                onChange={() => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, in_query: false }
                                                }))}
                                            />
                                            Send as HTTP Request Header
                                        </label>
                                        <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12.5px', cursor: 'pointer' }}>
                                            <input
                                                type="radio"
                                                name="key_location"
                                                checked={Boolean(outboundAuth.config.in_query)}
                                                onChange={() => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, in_query: true, query_param: outboundAuth.config.header_name || 'api_key' }
                                                }))}
                                            />
                                            Send as URL Query Parameter
                                        </label>
                                    </div>
                                </div>
                            )}

                            {/* AWS SigV4 Form */}
                            {outboundAuth.auth_type === 'aws-sigv4' && (
                                <div style={{ background: '#f8fafc', padding: '16px', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px', marginBottom: '12px' }}>
                                        <div>
                                            <label className="label">AWS Access Key ID *</label>
                                            <input
                                                type="text"
                                                className="input"
                                                placeholder="AKIAIOSFODNN7EXAMPLE"
                                                value={outboundAuth.config.access_key || ''}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, access_key: e.target.value }
                                                }))}
                                            />
                                        </div>
                                        <div>
                                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                                                <label className="label" style={{ margin: 0 }}>AWS Secret Access Key *</label>
                                                <button
                                                    type="button"
                                                    onClick={() => setShowOutboundSecret(!showOutboundSecret)}
                                                    style={{ background: 'none', border: 'none', color: '#2563eb', fontSize: '12px', cursor: 'pointer', padding: 0 }}
                                                >
                                                    {showOutboundSecret ? 'Hide' : 'Show'}
                                                </button>
                                            </div>
                                            <input
                                                type={showOutboundSecret ? 'text' : 'password'}
                                                className="input"
                                                placeholder={outboundAuth.config.secret_key_encrypted ? '•••••••• (Encrypted)' : 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY'}
                                                value={outboundAuth.config.secret_key || ''}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, secret_key: e.target.value }
                                                }))}
                                            />
                                        </div>
                                    </div>
                                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
                                        <div>
                                            <label className="label">AWS Region</label>
                                            <input
                                                type="text"
                                                className="input"
                                                placeholder="us-east-1"
                                                value={outboundAuth.config.region || 'us-east-1'}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, region: e.target.value }
                                                }))}
                                            />
                                        </div>
                                        <div>
                                            <label className="label">AWS Service</label>
                                            <input
                                                type="text"
                                                className="input"
                                                placeholder="execute-api"
                                                value={outboundAuth.config.service || 'execute-api'}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, service: e.target.value }
                                                }))}
                                            />
                                        </div>
                                    </div>
                                </div>
                            )}

                            {/* OAuth2 Client Credentials Form */}
                            {outboundAuth.auth_type === 'oauth2-cc' && (
                                <div style={{ background: '#f8fafc', padding: '16px', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
                                        <div>
                                            <label className="label">Client ID *</label>
                                            <input
                                                type="text"
                                                className="input"
                                                placeholder="e.g. gateway-client"
                                                value={outboundAuth.config.client_id || ''}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, client_id: e.target.value }
                                                }))}
                                            />
                                        </div>
                                        <div>
                                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                                                <label className="label" style={{ margin: 0 }}>Client Secret *</label>
                                                <button
                                                    type="button"
                                                    onClick={() => setShowOutboundSecret(!showOutboundSecret)}
                                                    style={{ background: 'none', border: 'none', color: '#2563eb', fontSize: '12px', cursor: 'pointer', padding: 0 }}
                                                >
                                                    {showOutboundSecret ? 'Hide' : 'Show'}
                                                </button>
                                            </div>
                                            <input
                                                type={showOutboundSecret ? 'text' : 'password'}
                                                className="input"
                                                placeholder={outboundAuth.config.client_secret_encrypted ? '•••••••• (Encrypted)' : 'OAuth2 client secret'}
                                                value={outboundAuth.config.client_secret || ''}
                                                onChange={e => setOutboundAuth(prev => ({
                                                    ...prev,
                                                    config: { ...prev.config, client_secret: e.target.value }
                                                }))}
                                            />
                                        </div>
                                    </div>
                                </div>
                            )}
                        </div>
                    )}
                </div>

                {/* Card 5: Inbound Auth Plugins (if editing existing service) */}
                {isEdit && id && (
                    <div style={{ marginBottom: '24px' }}>
                        <AuthPluginManager scope="service" entityId={id} entityName={formData.name} />
                    </div>
                )}

                {/* Bottom Sticky Action Bar */}
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
                        onClick={() => navigate('/services')}
                        disabled={isSaving}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="btn-header-new-route"
                        disabled={isSaving}
                    >
                        {isSaving ? 'Saving Service...' : isEdit ? 'Update Service' : 'Create Service'}
                    </button>
                </div>
            </form>
        </div>
    );
}
