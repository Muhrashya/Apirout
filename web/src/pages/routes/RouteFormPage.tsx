import { useState, useEffect } from 'react';
import { useNavigate, useParams, useSearchParams, Link } from 'react-router-dom';
import { gatewayApi, GatewayService, Route, QueryItem, GatewayDomain } from '../../api/gatewayApi';
import { useToast } from '../../components/Toast';
import { IconArrowLeft, IconServer, IconBolt, IconQuery, IconCheck, IconGlobe } from '../../components/Icons';

type TargetType = 'service' | 'instant_api' | 'query';

export default function RouteFormPage() {
    const { id } = useParams<{ id: string }>();
    const [searchParams] = useSearchParams();
    const isEdit = Boolean(id);
    const navigate = useNavigate();
    const toast = useToast();

    const [isLoading, setIsLoading] = useState(isEdit);
    const [isSaving, setIsSaving] = useState(false);
    const [services, setServices] = useState<GatewayService[]>([]);
    const [queries, setQueries] = useState<QueryItem[]>([]);
    const [registeredDomains, setRegisteredDomains] = useState<GatewayDomain[]>([]);
    const [targetType, setTargetType] = useState<TargetType>('service');

    const [formData, setFormData] = useState({
        name: '',
        path: '',
        method: 'GET',
        service_id: '',
        query_id: '',
        description: '',
        protocols: 'http,https',
        strip_path: false,
        preserve_host: false,
        rate_limit_enabled: false,
        rate_limit_rpm: 60,
        rate_limit_burst: 10,
        allowed_domains: '*',
        allowed_ips: '*',
        is_public: false,
        enabled: true
    });

    useEffect(() => {
        loadPrerequisites();
    }, [id]);

    const loadPrerequisites = async () => {
        try {
            const [servicesRes, queriesRes, domainsRes] = await Promise.all([
                gatewayApi.getServices().catch(() => ({ data: [] })),
                gatewayApi.getQueries().catch(() => ({ data: [] })),
                gatewayApi.getDomains().catch(() => ({ data: [] }))
            ]);

            const loadedServices = servicesRes.data || [];
            const loadedQueries = (queriesRes as any)?.data || (Array.isArray(queriesRes) ? queriesRes : []);
            const rawDomains = (domainsRes as any)?.data || (Array.isArray(domainsRes) ? domainsRes : []);
            const loadedDomains: GatewayDomain[] = Array.isArray(rawDomains) ? rawDomains : (rawDomains.data || []);

            setServices(loadedServices);
            setQueries(loadedQueries);
            setRegisteredDomains(loadedDomains);

            if (isEdit && id) {
                const routeRes = await gatewayApi.getRoute(id);
                const r: any = (routeRes as any)?.data || routeRes;
                if (r) {
                    if (r.query_id) {
                        setTargetType('query');
                    } else if (r.service_id) {
                        setTargetType('service');
                    } else {
                        setTargetType('instant_api');
                    }

                    setFormData({
                        name: r.name || '',
                        path: r.path || '',
                        method: r.method || 'GET',
                        service_id: r.service_id || '',
                        query_id: r.query_id || '',
                        description: r.description || '',
                        protocols: r.protocols || 'http,https',
                        strip_path: r.strip_path ?? false,
                        preserve_host: r.preserve_host ?? false,
                        rate_limit_enabled: r.rate_limit_enabled ?? false,
                        rate_limit_rpm: r.rate_limit_rpm || 60,
                        rate_limit_burst: r.rate_limit_burst || 10,
                        allowed_domains: r.allowed_domains || '*',
                        allowed_ips: r.allowed_ips || '*',
                        is_public: r.is_public ?? false,
                        enabled: r.enabled ?? true
                    });
                }
            } else {
                // Check query params for quick pre-selection
                const qId = searchParams.get('query_id');
                const sId = searchParams.get('service_id');
                if (qId) {
                    setTargetType('query');
                    setFormData(prev => ({ ...prev, query_id: qId }));
                } else if (sId) {
                    setTargetType('service');
                    setFormData(prev => ({ ...prev, service_id: sId }));
                }
            }
        } catch (err: any) {
            toast.error(err.message || 'Failed to load route data');
        } finally {
            setIsLoading(false);
        }
    };

    const handleSelectTargetType = (type: TargetType) => {
        setTargetType(type);
        if (type === 'service') {
            setFormData(prev => ({ ...prev, query_id: '' }));
        } else if (type === 'query') {
            setFormData(prev => ({ ...prev, service_id: '' }));
        } else if (type === 'instant_api') {
            setFormData(prev => ({ ...prev, service_id: '', query_id: '' }));
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        let cleanPath = formData.path.trim();
        if (!cleanPath.startsWith('/')) {
            cleanPath = '/' + cleanPath;
        }

        if (!cleanPath.startsWith('/api/')) {
            toast.error('Match Path must start with /api/ (e.g. /api/v1/resource)');
            return;
        }

        // Validate target selection according to active targetType
        if (targetType === 'service' && !formData.service_id) {
            toast.error('Please select a Target Gateway Service');
            return;
        }
        if (targetType === 'query' && !formData.query_id) {
            toast.error('Please select a Target SQL Query');
            return;
        }

        const payload: Partial<Route> = {
            name: formData.name.trim() || undefined,
            path: cleanPath,
            method: formData.method,
            service_id: targetType === 'service' ? (formData.service_id || undefined) : undefined,
            query_id: targetType === 'query' ? (formData.query_id || undefined) : undefined,
            description: formData.description.trim() || undefined,
            protocols: formData.protocols,
            strip_path: targetType === 'service' ? formData.strip_path : false,
            preserve_host: targetType === 'service' ? formData.preserve_host : false,
            rate_limit_enabled: formData.rate_limit_enabled,
            rate_limit_rpm: formData.rate_limit_rpm,
            rate_limit_burst: formData.rate_limit_burst,
            allowed_domains: formData.allowed_domains,
            allowed_ips: formData.allowed_ips,
            is_public: formData.is_public,
            enabled: formData.enabled
        };

        setIsSaving(true);
        try {
            if (isEdit && id) {
                await gatewayApi.updateRoute(id, payload);
                toast.success('Route updated successfully');
            } else {
                await gatewayApi.createRoute(payload);
                toast.success('Route created successfully');
            }
            navigate('/routes');
        } catch (err: any) {
            toast.error(err.message || 'Failed to save route');
        } finally {
            setIsSaving(false);
        }
    };

    const selectedDomainList = formData.allowed_domains
        ? formData.allowed_domains.split(',').map(d => d.trim()).filter(Boolean)
        : ['*'];
    const isWildcardDomain = selectedDomainList.includes('*') || selectedDomainList.length === 0;

    const toggleDomainSelection = (domainName: string) => {
        if (domainName === '*') {
            setFormData(prev => ({ ...prev, allowed_domains: '*' }));
            return;
        }

        if (isWildcardDomain) {
            setFormData(prev => ({ ...prev, allowed_domains: domainName }));
            return;
        }

        const exists = selectedDomainList.some(d => d.toLowerCase() === domainName.toLowerCase());
        let updated: string[];
        if (exists) {
            updated = selectedDomainList.filter(d => d.toLowerCase() !== domainName.toLowerCase());
            if (updated.length === 0) {
                updated = ['*'];
            }
        } else {
            updated = [...selectedDomainList.filter(d => d !== '*'), domainName];
        }

        setFormData(prev => ({ ...prev, allowed_domains: updated.join(', ') }));
    };

    if (isLoading) {
        return (
            <div style={{ padding: '60px', textAlign: 'center' }}>
                <div className="spinner" style={{ margin: '0 auto' }}></div>
                <div style={{ marginTop: '16px', color: '#64748b' }}>Loading route...</div>
            </div>
        );
    }

    return (
        <div style={{ maxWidth: '1080px', margin: '0', paddingBottom: '60px' }}>
            {/* Breadcrumb Header */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', color: '#64748b', marginBottom: '16px' }}>
                <span>Gateway</span>
                <span>/</span>
                <Link to="/routes" style={{ color: '#2563eb', textDecoration: 'none' }}>
                    Routes
                </Link>
                <span>/</span>
                <span style={{ color: '#0f172a', fontWeight: 600 }}>{isEdit ? 'Edit Route' : 'New Route'}</span>
            </div>

            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
                <div>
                    <h1 style={{ fontSize: '24px', fontWeight: 700, color: '#0f172a', margin: '0 0 6px' }}>
                        {isEdit ? 'Edit Gateway Route' : 'New Gateway Route'}
                    </h1>
                    <p style={{ fontSize: '14px', color: '#64748b', margin: 0 }}>
                        Kong-aligned route matching rules that bind client HTTP requests directly to backend services.
                    </p>
                </div>
                <Link to="/routes" className="btn-header-refresh" style={{ textDecoration: 'none' }}>
                    <IconArrowLeft width={14} height={14} />
                    <span>Back to Routes</span>
                </Link>
            </div>

            <form onSubmit={handleSubmit}>
                {/* Card 1: Route Matching Rules */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        1. Route Matching Rules
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: '140px 1fr', gap: '14px', marginBottom: '16px' }}>
                        <div>
                            <label className="label">HTTP Method *</label>
                            <select
                                className="select"
                                value={formData.method}
                                onChange={e => setFormData(prev => ({ ...prev, method: e.target.value }))}
                            >
                                <option value="GET">GET</option>
                                <option value="POST">POST</option>
                                <option value="PUT">PUT</option>
                                <option value="DELETE">DELETE</option>
                                <option value="PATCH">PATCH</option>
                            </select>
                        </div>
                        <div>
                            <label className="label">Match Path *</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="/api/v1/orders"
                                required
                                value={formData.path}
                                onChange={e => setFormData(prev => ({ ...prev, path: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Gateway routing path. Must start with <code>/api/</code>
                            </span>
                        </div>
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                        <div>
                            <label className="label">Route Name</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="e.g. orders-v1-get"
                                value={formData.name}
                                onChange={e => setFormData(prev => ({ ...prev, name: e.target.value }))}
                            />
                        </div>
                        <div>
                            <label className="label">Accepted Protocols</label>
                            <select
                                className="select"
                                value={formData.protocols}
                                onChange={e => setFormData(prev => ({ ...prev, protocols: e.target.value }))}
                            >
                                <option value="http,https">HTTP &amp; HTTPS</option>
                                <option value="http">HTTP Only</option>
                                <option value="https">HTTPS Only</option>
                                <option value="grpc">gRPC</option>
                            </select>
                        </div>
                    </div>

                    <div className="form-group">
                        <label className="label">Description</label>
                        <input
                            type="text"
                            className="input"
                            placeholder="Brief description of this route"
                            value={formData.description}
                            onChange={e => setFormData(prev => ({ ...prev, description: e.target.value }))}
                        />
                    </div>
                </div>

                {/* Card 2: Target Service Binding */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        <div>
                            <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 4px' }}>
                                2. Target Routing Destination
                            </h2>
                            <p style={{ fontSize: '13px', color: '#64748b', margin: 0 }}>
                                Choose whether requests are proxied to a Gateway Service, handled via Instant CRUD, or executed as a Database SQL Query.
                            </p>
                        </div>
                    </div>

                    {/* Target Type Selector Cards */}
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '14px', marginBottom: '20px' }}>
                        {/* Option 1: Gateway Service */}
                        <div
                            onClick={() => handleSelectTargetType('service')}
                            style={{
                                border: targetType === 'service' ? '2px solid #2563eb' : '1px solid #cbd5e1',
                                backgroundColor: targetType === 'service' ? '#eff6ff' : '#ffffff',
                                borderRadius: '10px',
                                padding: '16px',
                                cursor: 'pointer',
                                transition: 'all 150ms ease',
                                position: 'relative'
                            }}
                        >
                            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '8px' }}>
                                <span style={{
                                    width: '32px',
                                    height: '32px',
                                    borderRadius: '8px',
                                    backgroundColor: targetType === 'service' ? '#2563eb' : '#f1f5f9',
                                    color: targetType === 'service' ? '#ffffff' : '#64748b',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center'
                                }}>
                                    <IconServer width={18} height={18} />
                                </span>
                                <div>
                                    <div style={{ fontSize: '14px', fontWeight: 700, color: targetType === 'service' ? '#1e40af' : '#0f172a' }}>
                                        Gateway Service
                                    </div>
                                    <div style={{ fontSize: '11.5px', color: '#64748b' }}>
                                        HTTP Proxy / Upstream Pool
                                    </div>
                                </div>
                            </div>
                            <p style={{ fontSize: '12px', color: '#64748b', margin: 0, lineHeight: 1.4 }}>
                                Forward traffic to an existing microservice, backend host, or load-balanced upstream pool.
                            </p>
                            {targetType === 'service' && (
                                <div style={{ position: 'absolute', top: '12px', right: '12px', color: '#2563eb' }}>
                                    <IconCheck width={18} height={18} />
                                </div>
                            )}
                        </div>

                        {/* Option 2: Instant API */}
                        <div
                            onClick={() => handleSelectTargetType('instant_api')}
                            style={{
                                border: targetType === 'instant_api' ? '2px solid #2563eb' : '1px solid #cbd5e1',
                                backgroundColor: targetType === 'instant_api' ? '#eff6ff' : '#ffffff',
                                borderRadius: '10px',
                                padding: '16px',
                                cursor: 'pointer',
                                transition: 'all 150ms ease',
                                position: 'relative'
                            }}
                        >
                            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '8px' }}>
                                <span style={{
                                    width: '32px',
                                    height: '32px',
                                    borderRadius: '8px',
                                    backgroundColor: targetType === 'instant_api' ? '#2563eb' : '#f1f5f9',
                                    color: targetType === 'instant_api' ? '#ffffff' : '#64748b',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center'
                                }}>
                                    <IconBolt width={18} height={18} />
                                </span>
                                <div>
                                    <div style={{ fontSize: '14px', fontWeight: 700, color: targetType === 'instant_api' ? '#1e40af' : '#0f172a' }}>
                                        Instant API
                                    </div>
                                    <div style={{ fontSize: '11.5px', color: '#64748b' }}>
                                        Table CRUD Engine
                                    </div>
                                </div>
                            </div>
                            <p style={{ fontSize: '12px', color: '#64748b', margin: 0, lineHeight: 1.4 }}>
                                Automatically auto-generate REST endpoints against introspected database tables without writing code.
                            </p>
                            {targetType === 'instant_api' && (
                                <div style={{ position: 'absolute', top: '12px', right: '12px', color: '#2563eb' }}>
                                    <IconCheck width={18} height={18} />
                                </div>
                            )}
                        </div>

                        {/* Option 3: Database SQL Query */}
                        <div
                            onClick={() => handleSelectTargetType('query')}
                            style={{
                                border: targetType === 'query' ? '2px solid #2563eb' : '1px solid #cbd5e1',
                                backgroundColor: targetType === 'query' ? '#eff6ff' : '#ffffff',
                                borderRadius: '10px',
                                padding: '16px',
                                cursor: 'pointer',
                                transition: 'all 150ms ease',
                                position: 'relative'
                            }}
                        >
                            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '8px' }}>
                                <span style={{
                                    width: '32px',
                                    height: '32px',
                                    borderRadius: '8px',
                                    backgroundColor: targetType === 'query' ? '#2563eb' : '#f1f5f9',
                                    color: targetType === 'query' ? '#ffffff' : '#64748b',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center'
                                }}>
                                    <IconQuery width={18} height={18} />
                                </span>
                                <div>
                                    <div style={{ fontSize: '14px', fontWeight: 700, color: targetType === 'query' ? '#1e40af' : '#0f172a' }}>
                                        Database Query
                                    </div>
                                    <div style={{ fontSize: '11.5px', color: '#64748b' }}>
                                        Predefined SQL Query
                                    </div>
                                </div>
                            </div>
                            <p style={{ fontSize: '12px', color: '#64748b', margin: 0, lineHeight: 1.4 }}>
                                Execute predefined, parameterized SQL queries with custom joins, aggregations, and business logic.
                            </p>
                            {targetType === 'query' && (
                                <div style={{ position: 'absolute', top: '12px', right: '12px', color: '#2563eb' }}>
                                    <IconCheck width={18} height={18} />
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Sub-Panel: Gateway Service Configuration */}
                    {targetType === 'service' && (
                        <div style={{ padding: '16px', backgroundColor: '#f8fafc', borderRadius: '10px', border: '1px solid #e2e8f0' }}>
                            <div style={{ marginBottom: '14px' }}>
                                <label className="label">Target Gateway Service *</label>
                                <select
                                    className="select"
                                    required
                                    value={formData.service_id}
                                    onChange={e => setFormData(prev => ({ ...prev, service_id: e.target.value }))}
                                >
                                    <option value="">-- Select Gateway Service --</option>
                                    {services.map(s => (
                                        <option key={s.id} value={s.id}>
                                            {s.name} ({s.upstream_id ? 'Upstream Pool' : s.connection_id ? 'Database' : s.protocol})
                                        </option>
                                    ))}
                                </select>
                                <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                    Requests will be forwarded to this Service and its configured Upstream targets or Host URL.
                                </span>
                            </div>

                            <div style={{ display: 'flex', gap: '24px', flexWrap: 'wrap', paddingTop: '8px', borderTop: '1px solid #e2e8f0' }}>
                                <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', cursor: 'pointer', color: '#1e293b' }}>
                                    <input
                                        type="checkbox"
                                        checked={formData.strip_path}
                                        onChange={e => setFormData(prev => ({ ...prev, strip_path: e.target.checked }))}
                                        style={{ width: '16px', height: '16px' }}
                                    />
                                    <strong>Strip Path Prefix</strong> (removes route path before forwarding to backend)
                                </label>
                                <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', cursor: 'pointer', color: '#1e293b' }}>
                                    <input
                                        type="checkbox"
                                        checked={formData.preserve_host}
                                        onChange={e => setFormData(prev => ({ ...prev, preserve_host: e.target.checked }))}
                                        style={{ width: '16px', height: '16px' }}
                                    />
                                    <strong>Preserve Host</strong> (forward incoming Host header unchanged)
                                </label>
                            </div>
                        </div>
                    )}

                    {/* Sub-Panel: Instant API Information */}
                    {targetType === 'instant_api' && (
                        <div style={{ padding: '20px', backgroundColor: '#f0fdf4', borderRadius: '10px', border: '1px solid #bbf7d0' }}>
                            <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px' }}>
                                <div style={{ color: '#16a34a', marginTop: '2px' }}>
                                    <IconBolt width={22} height={22} />
                                </div>
                                <div style={{ flex: 1 }}>
                                    <h4 style={{ fontSize: '14px', fontWeight: 600, color: '#166534', margin: '0 0 6px' }}>
                                        Instant API Table Routing Active
                                    </h4>
                                    <p style={{ fontSize: '13px', color: '#15803d', margin: '0 0 12px', lineHeight: 1.5 }}>
                                        Incoming HTTP requests matching table paths (e.g. <code>/api/v1/customers</code>) will be dispatched directly to the built-in Instant API database engine.
                                    </p>
                                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                        <Link
                                            to="/instant-api"
                                            target="_blank"
                                            style={{
                                                fontSize: '12.5px',
                                                fontWeight: 600,
                                                color: '#15803d',
                                                textDecoration: 'none',
                                                backgroundColor: '#dcfce7',
                                                padding: '6px 12px',
                                                borderRadius: '6px',
                                                display: 'inline-flex',
                                                alignItems: 'center',
                                                gap: '6px'
                                            }}
                                        >
                                            <span>Manage Instant API Schemas &rarr;</span>
                                        </Link>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Sub-Panel: Database SQL Query Configuration */}
                    {targetType === 'query' && (
                        <div style={{ padding: '16px', backgroundColor: '#f8fafc', borderRadius: '10px', border: '1px solid #e2e8f0' }}>
                            <div style={{ marginBottom: '14px' }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                                    <label className="label" style={{ margin: 0 }}>Target Database SQL Query *</label>
                                    <Link to="/queries" target="_blank" style={{ fontSize: '12px', color: '#2563eb', textDecoration: 'none', fontWeight: 500 }}>
                                        + Open Queries Workbench
                                    </Link>
                                </div>
                                <select
                                    className="select"
                                    required
                                    value={formData.query_id}
                                    onChange={e => setFormData(prev => ({ ...prev, query_id: e.target.value }))}
                                >
                                    <option value="">-- Select Saved SQL Query --</option>
                                    {queries.map(q => (
                                        <option key={q.id} value={q.id}>
                                            {q.name}
                                        </option>
                                    ))}
                                </select>
                            </div>

                            {(() => {
                                const selectedQ = queries.find(q => q.id === formData.query_id);
                                if (!selectedQ) return null;
                                return (
                                    <div style={{ marginTop: '12px', padding: '12px', backgroundColor: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '8px' }}>
                                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
                                            <span style={{ fontSize: '11.5px', fontWeight: 600, color: '#475569', textTransform: 'uppercase' }}>
                                                Selected Query:
                                            </span>
                                            <strong style={{ fontSize: '13px', color: '#0f172a' }}>{selectedQ.name}</strong>
                                        </div>
                                        <div style={{ backgroundColor: '#0f172a', padding: '10px 12px', borderRadius: '6px', overflowX: 'auto' }}>
                                            <code style={{ fontFamily: 'monospace', fontSize: '12px', color: '#38bdf8', whiteSpace: 'pre-wrap' }}>
                                                {selectedQ.sql_text}
                                            </code>
                                        </div>
                                    </div>
                                );
                            })()}
                        </div>
                    )}
                </div>

                {/* Card 3: Security & Rate Limiting */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        3. Security, Access &amp; Rate Limiting
                    </h2>

                    <div style={{ marginBottom: '16px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '10px' }}>
                            <span style={{ fontSize: '13.5px', fontWeight: 600, color: '#0f172a' }}>Rate Limiting Policy</span>
                            <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', cursor: 'pointer' }}>
                                <input
                                    type="checkbox"
                                    checked={formData.rate_limit_enabled}
                                    onChange={e => setFormData(prev => ({ ...prev, rate_limit_enabled: e.target.checked }))}
                                    style={{ width: '15px', height: '15px' }}
                                />
                                Enable Specific Route Rate Limit
                            </label>
                        </div>

                        {formData.rate_limit_enabled && (
                            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', padding: '14px', background: '#f8fafc', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                                <div>
                                    <label className="label">Requests Per Minute (RPM)</label>
                                    <input
                                        type="number"
                                        className="input"
                                        min={1}
                                        value={formData.rate_limit_rpm}
                                        onChange={e => setFormData(prev => ({ ...prev, rate_limit_rpm: parseInt(e.target.value) || 60 }))}
                                    />
                                </div>
                                <div>
                                    <label className="label">Burst Allowance</label>
                                    <input
                                        type="number"
                                        className="input"
                                        min={1}
                                        value={formData.rate_limit_burst}
                                        onChange={e => setFormData(prev => ({ ...prev, rate_limit_burst: parseInt(e.target.value) || 10 }))}
                                    />
                                </div>
                            </div>
                        )}
                    </div>

                    {/* Host-based Routing / Gateway Domains Integration */}
                    <div style={{
                        backgroundColor: '#f8fafc',
                        border: '1px solid #e2e8f0',
                        borderRadius: '10px',
                        padding: '16px',
                        marginBottom: '16px'
                    }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '10px', flexWrap: 'wrap', gap: '8px' }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <IconGlobe width={16} height={16} style={{ color: '#2563eb' }} />
                                <label className="label" style={{ margin: 0, fontWeight: 600 }}>Host-based Routing / Allowed Domains</label>
                            </div>
                            <Link
                                to="/settings/domains"
                                target="_blank"
                                style={{
                                    fontSize: '12px',
                                    color: '#2563eb',
                                    textDecoration: 'none',
                                    fontWeight: 500,
                                    display: 'inline-flex',
                                    alignItems: 'center',
                                    gap: '4px'
                                }}
                            >
                                + Manage Gateway Domains ↗
                            </Link>
                        </div>
                        <p style={{ fontSize: '12.5px', color: '#64748b', margin: '0 0 12px 0', lineHeight: 1.4 }}>
                            Select registered Gateway domains to bind this route to specific hostnames, or type custom wildcard hosts. Incoming requests matching this route must have a matching <code>Host</code> header.
                        </p>

                        {/* Registered Domains Pill Selector */}
                        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginBottom: '14px' }}>
                            <button
                                type="button"
                                onClick={() => toggleDomainSelection('*')}
                                style={{
                                    display: 'inline-flex',
                                    alignItems: 'center',
                                    gap: '6px',
                                    padding: '5px 12px',
                                    borderRadius: '6px',
                                    fontSize: '12.5px',
                                    fontWeight: 600,
                                    cursor: 'pointer',
                                    transition: 'all 150ms ease',
                                    border: isWildcardDomain ? '1.5px solid #2563eb' : '1px solid #cbd5e1',
                                    backgroundColor: isWildcardDomain ? '#eff6ff' : '#ffffff',
                                    color: isWildcardDomain ? '#1d4ed8' : '#475569'
                                }}
                            >
                                <span>*</span>
                                <span>Any Domain (Wildcard)</span>
                                {isWildcardDomain && <IconCheck width={13} height={13} style={{ color: '#2563eb' }} />}
                            </button>

                            {registeredDomains.map(dom => {
                                const isSelected = !isWildcardDomain && selectedDomainList.some(d => d.toLowerCase() === dom.domain_name.toLowerCase());
                                return (
                                    <button
                                        key={dom.id}
                                        type="button"
                                        onClick={() => toggleDomainSelection(dom.domain_name)}
                                        style={{
                                            display: 'inline-flex',
                                            alignItems: 'center',
                                            gap: '6px',
                                            padding: '5px 12px',
                                            borderRadius: '6px',
                                            fontSize: '12.5px',
                                            fontWeight: 500,
                                            cursor: 'pointer',
                                            transition: 'all 150ms ease',
                                            border: isSelected ? '1.5px solid #2563eb' : '1px solid #cbd5e1',
                                            backgroundColor: isSelected ? '#eff6ff' : '#ffffff',
                                            color: isSelected ? '#1d4ed8' : '#334155'
                                        }}
                                        title={`${dom.display_name} (${dom.environment})`}
                                    >
                                        <span style={{ fontFamily: 'monospace', fontWeight: isSelected ? 700 : 500 }}>
                                            {dom.domain_name}
                                        </span>
                                        {dom.is_default && (
                                            <span style={{
                                                fontSize: '10px',
                                                padding: '1px 5px',
                                                borderRadius: '4px',
                                                backgroundColor: isSelected ? '#bfdbfe' : '#f1f5f9',
                                                color: isSelected ? '#1e40af' : '#64748b',
                                                fontWeight: 600
                                            }}>
                                                Default
                                            </span>
                                        )}
                                        <span style={{
                                            fontSize: '10px',
                                            padding: '1px 5px',
                                            borderRadius: '4px',
                                            backgroundColor: dom.environment === 'production' ? '#dcfce7' : '#fef3c7',
                                            color: dom.environment === 'production' ? '#166534' : '#92400e',
                                            textTransform: 'uppercase',
                                            fontWeight: 600
                                        }}>
                                            {dom.environment.slice(0, 4)}
                                        </span>
                                        {isSelected && <IconCheck width={13} height={13} style={{ color: '#2563eb' }} />}
                                    </button>
                                );
                            })}
                        </div>

                        {/* Raw Input for fine-grained editing / custom wildcard hosts */}
                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '14px' }}>
                            <div>
                                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#475569', marginBottom: '4px' }}>
                                    Active Allowed Domains Pattern
                                </label>
                                <input
                                    type="text"
                                    className="input"
                                    value={formData.allowed_domains}
                                    onChange={e => setFormData(prev => ({ ...prev, allowed_domains: e.target.value }))}
                                    placeholder="* or api.example.com, *.example.com"
                                    style={{ fontSize: '13px', fontFamily: 'monospace' }}
                                />
                                <span style={{ fontSize: '11px', color: '#64748b', marginTop: '3px', display: 'block' }}>
                                    Comma-separated domains or wildcards (e.g. <code>*.acme.com</code>).
                                </span>
                            </div>
                            <div>
                                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#475569', marginBottom: '4px' }}>
                                    Allowed Client IPs
                                </label>
                                <input
                                    type="text"
                                    className="input"
                                    value={formData.allowed_ips}
                                    onChange={e => setFormData(prev => ({ ...prev, allowed_ips: e.target.value }))}
                                    placeholder="*"
                                    style={{ fontSize: '13px', fontFamily: 'monospace' }}
                                />
                                <span style={{ fontSize: '11px', color: '#64748b', marginTop: '3px', display: 'block' }}>
                                    IP whitelist / CIDR blocks (e.g. <code>192.168.1.0/24</code> or <code>*</code>).
                                </span>
                            </div>
                        </div>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <input
                            type="checkbox"
                            id="route-public"
                            checked={formData.is_public}
                            onChange={e => setFormData(prev => ({ ...prev, is_public: e.target.checked }))}
                            style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                        />
                        <label htmlFor="route-public" style={{ fontSize: '13.5px', fontWeight: 500, color: '#1e293b', cursor: 'pointer' }}>
                            Public Route (skip API Key / Consumer authentication requirements)
                        </label>
                    </div>
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
                        onClick={() => navigate('/routes')}
                        disabled={isSaving}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="btn-header-new-route"
                        disabled={isSaving}
                    >
                        {isSaving ? 'Saving Route...' : isEdit ? 'Update Route' : 'Create Route'}
                    </button>
                </div>
            </form>
        </div>
    );
}
