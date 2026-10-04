import { useState, useEffect } from 'react';
import { gatewayApi, ApiRequest } from '../../api/gatewayApi';
import { useToast } from '../../components/Toast';
import {
    IconRefresh, IconSearch, IconClose, IconCopy, IconCheck
} from '../../components/Icons';

export default function ApiLogsPage() {
    const [requests, setRequests] = useState<ApiRequest[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const [statusFilter, setStatusFilter] = useState('all');
    const [methodFilter, setMethodFilter] = useState('all');

    // Selected Request for Inspector Drawer
    const [selectedReq, setSelectedReq] = useState<ApiRequest | null>(null);
    const [drawerTab, setDrawerTab] = useState<'general' | 'service' | 'request' | 'response'>('general');
    const [isCopied, setIsCopied] = useState(false);

    const toast = useToast();

    useEffect(() => {
        loadRequests();
    }, []);

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape' && selectedReq) {
                setSelectedReq(null);
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [selectedReq]);

    const loadRequests = async () => {
        setIsLoading(true);
        try {
            const res = await gatewayApi.getRequests(100);
            setRequests(res.data || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load request logs');
        } finally {
            setIsLoading(false);
        }
    };

    const handleCopyCurl = (req: ApiRequest) => {
        const curlCmd = `curl -X ${req.method} "http://localhost:8000${req.path}"`;
        navigator.clipboard.writeText(curlCmd);
        setIsCopied(true);
        toast.success('cURL command copied to clipboard');
        setTimeout(() => setIsCopied(false), 2000);
    };

    const filteredRequests = requests.filter(r => {
        if (searchTerm) {
            const term = searchTerm.toLowerCase();
            const matchesPath = r.path?.toLowerCase().includes(term);
            const matchesClient = r.client_ip?.toLowerCase().includes(term);
            const matchesUser = r.username?.toLowerCase().includes(term);
            if (!matchesPath && !matchesClient && !matchesUser) return false;
        }

        if (methodFilter !== 'all' && r.method !== methodFilter) {
            return false;
        }

        if (statusFilter !== 'all') {
            const code = r.status_code || 200;
            if (statusFilter === '2xx' && (code < 200 || code >= 300)) return false;
            if (statusFilter === '3xx' && (code < 300 || code >= 400)) return false;
            if (statusFilter === '4xx' && (code < 400 || code >= 500)) return false;
            if (statusFilter === '5xx' && code < 500) return false;
        }

        return true;
    });

    const getStatusBadge = (code: number) => {
        let bg = '#eff6ff';
        let color = '#2563eb';
        if (code >= 200 && code < 300) {
            bg = '#dcfce7';
            color = '#16a34a';
        } else if (code >= 300 && code < 400) {
            bg = '#f1f5f9';
            color = '#475569';
        } else if (code === 401 || code === 403) {
            bg = '#fef9c3';
            color = '#a16207';
        } else if (code === 429) {
            bg = '#ffedd5';
            color = '#c2410c';
        } else if (code >= 400) {
            bg = '#fee2e2';
            color = '#dc2626';
        }

        return (
            <span style={{
                display: 'inline-block',
                minWidth: '42px',
                textAlign: 'center',
                padding: '2px 8px',
                borderRadius: '6px',
                fontSize: '12px',
                fontWeight: 700,
                backgroundColor: bg,
                color: color
            }}>
                {code || 200}
            </span>
        );
    };

    const getMethodBadge = (m: string) => {
        let color = '#2563eb';
        if (m === 'POST') color = '#16a34a';
        if (m === 'PUT') color = '#d97706';
        if (m === 'DELETE') color = '#dc2626';
        if (m === 'PATCH') color = '#9333ea';

        return (
            <span style={{
                fontSize: '11px',
                fontWeight: 700,
                color: color,
                fontFamily: 'monospace'
            }}>
                {m}
            </span>
        );
    };

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', position: 'relative' }}>
            {/* Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', color: '#64748b', marginBottom: '4px' }}>
                        <span>Observability</span>
                        <span>/</span>
                        <span style={{ color: '#0f172a', fontWeight: 600 }}>API Logs</span>
                    </div>
                    <h1 style={{ fontSize: '24px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                        Requests &amp; API Logs
                    </h1>
                </div>

                <div style={{ display: 'flex', gap: '10px' }}>
                    <button className="btn-header-refresh" onClick={loadRequests} disabled={isLoading}>
                        <IconRefresh width={14} height={14} />
                        <span>Refresh</span>
                    </button>
                </div>
            </div>

            {/* Filters Bar */}
            <div style={{
                background: '#ffffff',
                border: '1px solid #e2e8f0',
                borderRadius: '12px',
                padding: '12px 18px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                flexWrap: 'wrap',
                gap: '12px',
                boxShadow: '0 1px 3px rgba(0,0,0,0.04)'
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
                    {/* Search Input */}
                    <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        background: '#f8fafc',
                        border: '1px solid #e2e8f0',
                        borderRadius: '8px',
                        padding: '0 12px',
                        width: '320px',
                        height: '38px'
                    }}>
                        <IconSearch width={15} height={15} style={{ color: '#94a3b8', marginRight: '8px' }} />
                        <input
                            type="text"
                            placeholder="Filter by path, IP, or consumer..."
                            value={searchTerm}
                            onChange={e => setSearchTerm(e.target.value)}
                            style={{ border: 'none', background: 'transparent', fontSize: '13px', outline: 'none', width: '100%', color: '#0f172a' }}
                        />
                    </div>

                    {/* Method Filter */}
                    <select
                        className="select select-filter"
                        value={methodFilter}
                        onChange={e => setMethodFilter(e.target.value)}
                    >
                        <option value="all">All Methods</option>
                        <option value="GET">GET</option>
                        <option value="POST">POST</option>
                        <option value="PUT">PUT</option>
                        <option value="DELETE">DELETE</option>
                        <option value="PATCH">PATCH</option>
                    </select>

                    {/* Status Filter */}
                    <select
                        className="select select-filter"
                        value={statusFilter}
                        onChange={e => setStatusFilter(e.target.value)}
                    >
                        <option value="all">All Status Codes</option>
                        <option value="2xx">2xx Success</option>
                        <option value="3xx">3xx Redirection</option>
                        <option value="4xx">4xx Client Errors</option>
                        <option value="5xx">5xx Server Errors</option>
                    </select>
                </div>

                <div style={{ fontSize: '13px', color: '#64748b' }}>
                    Showing <strong>{filteredRequests.length}</strong> recorded requests
                </div>
            </div>

            {/* Requests Table */}
            <div style={{
                background: '#ffffff',
                border: '1px solid #e2e8f0',
                borderRadius: '12px',
                overflow: 'hidden',
                boxShadow: '0 1px 3px rgba(0,0,0,0.04)'
            }}>
                <div style={{ overflowX: 'auto' }}>
                    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
                        <thead>
                            <tr style={{ background: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                <th style={{ padding: '12px 16px', fontWeight: 600, color: '#475569' }}>Timestamp</th>
                                <th style={{ padding: '12px 16px', fontWeight: 600, color: '#475569' }}>Status</th>
                                <th style={{ padding: '12px 16px', fontWeight: 600, color: '#475569' }}>Method</th>
                                <th style={{ padding: '12px 16px', fontWeight: 600, color: '#475569' }}>Path</th>
                                <th style={{ padding: '12px 16px', fontWeight: 600, color: '#475569' }}>Latency</th>
                                <th style={{ padding: '12px 16px', fontWeight: 600, color: '#475569' }}>Consumer</th>
                                <th style={{ padding: '12px 16px', fontWeight: 600, color: '#475569' }}>Client IP</th>
                            </tr>
                        </thead>
                        <tbody>
                            {isLoading ? (
                                <tr>
                                    <td colSpan={7} style={{ padding: '40px', textAlign: 'center' }}>
                                        <div className="spinner" style={{ margin: '0 auto' }}></div>
                                        <div style={{ marginTop: '12px', color: '#64748b' }}>Loading logs...</div>
                                    </td>
                                </tr>
                            ) : filteredRequests.length === 0 ? (
                                <tr>
                                    <td colSpan={7} style={{ padding: '40px', textAlign: 'center', color: '#94a3b8' }}>
                                        No API requests matched the filter criteria.
                                    </td>
                                </tr>
                            ) : (
                                filteredRequests.map(req => (
                                    <tr
                                        key={req.id}
                                        onClick={() => setSelectedReq(req)}
                                        style={{
                                            borderBottom: '1px solid #f1f5f9',
                                            cursor: 'pointer',
                                            transition: 'background 120ms',
                                            background: selectedReq?.id === req.id ? '#eff6ff' : '#ffffff'
                                        }}
                                        onMouseEnter={e => { if (selectedReq?.id !== req.id) e.currentTarget.style.backgroundColor = '#f8fafc'; }}
                                        onMouseLeave={e => { if (selectedReq?.id !== req.id) e.currentTarget.style.backgroundColor = '#ffffff'; }}
                                    >
                                        <td style={{ padding: '12px 16px', color: '#64748b', whiteSpace: 'nowrap' }}>
                                            {new Date(req.timestamp).toLocaleString()}
                                        </td>
                                        <td style={{ padding: '12px 16px' }}>
                                            {getStatusBadge(req.status_code)}
                                        </td>
                                        <td style={{ padding: '12px 16px' }}>
                                            {getMethodBadge(req.method)}
                                        </td>
                                        <td style={{ padding: '12px 16px', fontWeight: 600, color: '#0f172a', fontFamily: 'monospace' }}>
                                            {req.path}
                                        </td>
                                        <td style={{ padding: '12px 16px', color: '#475569' }}>
                                            {req.duration_ms !== null && req.duration_ms !== undefined ? `${req.duration_ms}ms` : '—'}
                                        </td>
                                        <td style={{ padding: '12px 16px', color: '#334155' }}>
                                            {req.username || 'Anonymous'}
                                        </td>
                                        <td style={{ padding: '12px 16px', color: '#64748b', fontFamily: 'monospace' }}>
                                            {req.client_ip || '—'}
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* ======================================================== */}
            {/* SLIDE-OVER REQUEST INSPECTOR DRAWER (MATCHING KONG IMAGE 2) */}
            {/* ======================================================== */}
            {selectedReq && (
                <>
                    {/* Gray transparent backdrop overlay */}
                    <div
                        onClick={() => setSelectedReq(null)}
                        style={{
                            position: 'fixed',
                            top: 0,
                            left: 0,
                            right: 0,
                            bottom: 0,
                            backgroundColor: 'rgba(15, 23, 42, 0.45)',
                            backdropFilter: 'blur(2px)',
                            WebkitBackdropFilter: 'blur(2px)',
                            zIndex: 99,
                            cursor: 'pointer',
                            animation: 'drawerOverlayFadeIn 220ms ease-out forwards'
                        }}
                    />

                    {/* Slide-over Drawer with Smooth Slide Animation */}
                    <div style={{
                        position: 'fixed',
                        top: 0,
                        right: 0,
                        bottom: 0,
                        width: '620px',
                        maxWidth: '90vw',
                        background: '#ffffff',
                        boxShadow: '-8px 0 32px rgba(15, 23, 42, 0.2)',
                        zIndex: 100,
                        display: 'flex',
                        flexDirection: 'column',
                        animation: 'drawerSlideInRight 260ms cubic-bezier(0.16, 1, 0.3, 1) forwards'
                    }}>
                    {/* Drawer Header (Status + Path + Actions) */}
                    <div style={{
                        padding: '18px 24px',
                        borderBottom: '1px solid #e2e8f0',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        background: '#f8fafc'
                    }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', overflow: 'hidden' }}>
                            {getStatusBadge(selectedReq.status_code)}
                            <span style={{ fontSize: '15px', fontWeight: 700, color: '#0f172a', fontFamily: 'monospace', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>
                                {selectedReq.method} {selectedReq.path}
                            </span>
                        </div>

                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <button
                                type="button"
                                onClick={() => handleCopyCurl(selectedReq)}
                                title="Copy cURL"
                                style={{ border: '1px solid #cbd5e1', background: '#ffffff', padding: '6px', borderRadius: '6px', cursor: 'pointer', color: '#475569' }}
                            >
                                {isCopied ? <IconCheck width={15} height={15} style={{ color: '#16a34a' }} /> : <IconCopy width={15} height={15} />}
                            </button>
                            <button
                                type="button"
                                onClick={() => setSelectedReq(null)}
                                title="Close Inspector"
                                style={{ border: 'none', background: 'transparent', padding: '6px', borderRadius: '6px', cursor: 'pointer', color: '#64748b' }}
                            >
                                <IconClose width={18} height={18} />
                            </button>
                        </div>
                    </div>

                    {/* Latency & Timestamp Summary Bar */}
                    <div style={{
                        padding: '10px 24px',
                        background: '#f1f5f9',
                        borderBottom: '1px solid #e2e8f0',
                        display: 'flex',
                        justifyContent: 'space-between',
                        fontSize: '12px',
                        color: '#475569'
                    }}>
                        <div style={{ display: 'flex', gap: '16px' }}>
                            <span><strong>Response:</strong> {selectedReq.duration_ms ?? 0}ms</span>
                            <span><strong>Gateway:</strong> {selectedReq.gateway_latency_ms ?? (selectedReq.duration_ms ?? 0)}ms</span>
                            <span><strong>Upstream:</strong> {selectedReq.upstream_latency_ms ?? 0}ms</span>
                        </div>
                        <div>
                            <span><strong>Time:</strong> {new Date(selectedReq.timestamp).toLocaleTimeString()}</span>
                        </div>
                    </div>

                    {/* Navigation Tabs (General, Service, Request, Response) */}
                    <div style={{ display: 'flex', borderBottom: '1px solid #e2e8f0', background: '#ffffff' }}>
                        {(['general', 'service', 'request', 'response'] as const).map(tab => (
                            <button
                                key={tab}
                                onClick={() => setDrawerTab(tab)}
                                style={{
                                    flex: 1,
                                    padding: '12px 14px',
                                    fontSize: '13px',
                                    fontWeight: drawerTab === tab ? 700 : 500,
                                    color: drawerTab === tab ? '#2563eb' : '#64748b',
                                    borderBottom: drawerTab === tab ? '2px solid #2563eb' : 'none',
                                    borderTop: 'none',
                                    borderLeft: 'none',
                                    borderRight: 'none',
                                    background: 'transparent',
                                    cursor: 'pointer',
                                    textTransform: 'capitalize'
                                }}
                            >
                                {tab}
                            </button>
                        ))}
                    </div>

                    {/* Tab Contents */}
                    <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>
                        {drawerTab === 'general' && (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                                <InspectorRow label="Client IP" value={selectedReq.client_ip || '127.0.0.1'} mono />
                                <InspectorRow label="Upstream URI" value={selectedReq.upstream_uri || selectedReq.path} mono />
                                <InspectorRow label="Auth Type" value={selectedReq.auth_type || (selectedReq.api_client_id ? 'key-auth' : 'anonymous')} badge />
                                <InspectorRow label="Control Plane" value="apirout-cp-primary" />
                                <InspectorRow label="Data Plane Node" value="apirout-runtime-dp1" />
                                <InspectorRow label="Route" value={selectedReq.route_name || selectedReq.endpoint_id || 'Direct Route'} link="/routes" />
                                <InspectorRow label="Service" value={selectedReq.service_name || 'Core Gateway Service'} link="/services" />
                                <InspectorRow label="Consumer" value={selectedReq.username || 'Anonymous'} link="/consumers" />
                            </div>
                        )}

                        {drawerTab === 'service' && (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                                <InspectorRow label="Service Name" value={selectedReq.service_name || 'Gateway Core Service'} />
                                <InspectorRow label="Upstream Destination" value={selectedReq.upstream_uri || 'http://localhost:8000'} mono />
                                <InspectorRow label="Protocol" value="HTTP/1.1" />
                                <InspectorRow label="Upstream Latency" value={`${selectedReq.upstream_latency_ms ?? 0}ms`} />
                                <InspectorRow label="Health Status" value="Healthy" badge />
                            </div>
                        )}

                        {drawerTab === 'request' && (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                                <InspectorRow label="User Agent" value={selectedReq.user_agent || 'Mozilla/5.0'} />
                                <InspectorRow label="Host" value={selectedReq.host || 'localhost:8000'} mono />
                                <InspectorRow label="Method" value={selectedReq.method} badge />
                                <InspectorRow label="URI" value={selectedReq.path} mono />
                                <InspectorRow label="Payload Size" value={`${selectedReq.size || 0} B`} />
                                <InspectorRow label="Client IP" value={selectedReq.client_ip || '127.0.0.1'} mono />
                            </div>
                        )}

                        {drawerTab === 'response' && (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                                <InspectorRow label="Status Code" value={String(selectedReq.status_code || 200)} badge />
                                <InspectorRow label="Duration" value={`${selectedReq.duration_ms ?? 0}ms`} />
                                <InspectorRow label="Content-Type" value="application/json" mono />
                                {selectedReq.error_message && (
                                    <div style={{ background: '#fee2e2', border: '1px solid #fca5a5', borderRadius: '8px', padding: '14px', marginTop: '10px' }}>
                                        <div style={{ fontWeight: 700, color: '#dc2626', fontSize: '13px', marginBottom: '4px' }}>Error Details</div>
                                        <code style={{ color: '#991b1b', fontSize: '12.5px' }}>{selectedReq.error_message}</code>
                                    </div>
                                )}
                            </div>
                        )}
                    </div>
                </div>
                </>
            )}

            {/* Injected CSS keyframes for smooth drawer and overlay transitions */}
            <style>{`
                @keyframes drawerSlideInRight {
                    from {
                        transform: translateX(100%);
                        opacity: 0.6;
                    }
                    to {
                        transform: translateX(0);
                        opacity: 1;
                    }
                }
                @keyframes drawerOverlayFadeIn {
                    from {
                        opacity: 0;
                    }
                    to {
                        opacity: 1;
                    }
                }
            `}</style>
        </div>
    );
}

function InspectorRow({ label, value, mono, badge, link }: { label: string; value: string; mono?: boolean; badge?: boolean; link?: string }) {
    return (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingBottom: '12px', borderBottom: '1px solid #f1f5f9' }}>
            <span style={{ fontSize: '13px', color: '#64748b', fontWeight: 500 }}>{label}</span>
            {badge ? (
                <span style={{ fontSize: '12px', fontWeight: 600, padding: '2px 8px', borderRadius: '4px', background: '#eff6ff', color: '#2563eb' }}>
                    {value}
                </span>
            ) : mono ? (
                <code style={{ fontSize: '12.5px', color: '#0f172a', fontWeight: 600, background: '#f8fafc', padding: '2px 6px', borderRadius: '4px' }}>
                    {value}
                </code>
            ) : (
                <span style={{ fontSize: '13.5px', color: link ? '#2563eb' : '#0f172a', fontWeight: link ? 600 : 400 }}>
                    {value}
                </span>
            )}
        </div>
    );
}
