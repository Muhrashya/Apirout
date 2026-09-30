import { useState, useEffect } from 'react';
import { apiClient } from '../../api/apiClient';
import { useToast } from '../../components/Toast';
import {
    IconDownload, IconCopy, IconCheck, IconSearch,
    IconBolt, IconFileText, IconBook, IconKey
} from '../../components/Icons';

interface Endpoint {
    id: string;
    path: string;
    method: string;
    summary?: string;
    description?: string;
}

export default function ApiCollections() {
    const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
    const [isLoading, setIsLoading] = useState(true);
    const [searchQuery, setSearchQuery] = useState('');
    const [copiedKey, setCopiedKey] = useState<string | null>(null);
    const [activeAuthTab, setActiveAuthTab] = useState<'bearer' | 'apiKey' | 'clientLogin' | 'adminLogin'>('bearer');
    const toast = useToast();

    useEffect(() => {
        loadOpenAPI();
    }, []);

    const loadOpenAPI = async () => {
        try {
            const spec = await apiClient.get<{ paths: Record<string, Record<string, any>> }>('/openapi.json');
            const paths = spec.paths || {};
            const eps = Object.entries(paths).flatMap(([path, methods]) =>
                Object.entries(methods).map(([method, details]) => ({
                    id: details.operationId || `${method.toUpperCase()}:${path}`,
                    path,
                    method: method.toUpperCase(),
                    summary: details.summary || '',
                    description: details.description || ''
                }))
            );
            setEndpoints(eps);
        } catch (err) {
            toast.error('Failed to load API endpoints for collections');
        } finally {
            setIsLoading(false);
        }
    };

    const copyToClipboard = (text: string, key: string, label: string) => {
        navigator.clipboard.writeText(text);
        setCopiedKey(key);
        toast.success(`Copied ${label} to clipboard`);
        setTimeout(() => setCopiedKey(null), 2000);
    };

    const toggleSelectAll = (checked: boolean) => {
        if (checked) {
            setSelectedIds(new Set(filteredEndpoints.map(ep => ep.id)));
        } else {
            setSelectedIds(new Set());
        }
    };

    const toggleSelect = (id: string) => {
        const next = new Set(selectedIds);
        if (next.has(id)) {
            next.delete(id);
        } else {
            next.add(id);
        }
        setSelectedIds(next);
    };

    const getDownloadQuery = () => {
        if (selectedIds.size === 0 || selectedIds.size === endpoints.length) {
            return '';
        }
        return `?endpoint_ids=${Array.from(selectedIds).join(',')}`;
    };

    const downloadPostman = async () => {
        try {
            const query = getDownloadQuery();
            const collection = await apiClient.get<unknown>(`/postman.json${query}`);
            const blob = new Blob([JSON.stringify(collection, null, 2)], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `apirout-collection${selectedIds.size > 0 && selectedIds.size < endpoints.length ? '-selected' : ''}.json`;
            a.click();
            URL.revokeObjectURL(url);
            toast.success('Postman collection v2.1 downloaded successfully');
        } catch (err) {
            toast.error('Failed to download Postman collection');
        }
    };

    const downloadOpenAPI = async () => {
        try {
            const query = getDownloadQuery();
            const spec = await apiClient.get<unknown>(`/openapi.json${query}`);
            const blob = new Blob([JSON.stringify(spec, null, 2)], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `apirout-openapi${selectedIds.size > 0 && selectedIds.size < endpoints.length ? '-selected' : ''}.json`;
            a.click();
            URL.revokeObjectURL(url);
            toast.success('OpenAPI 3.0 specification downloaded successfully');
        } catch (err) {
            toast.error('Failed to download OpenAPI spec');
        }
    };

    const filteredEndpoints = endpoints.filter(ep => {
        const query = searchQuery.toLowerCase();
        return ep.path.toLowerCase().includes(query) ||
            ep.method.toLowerCase().includes(query) ||
            (ep.summary && ep.summary.toLowerCase().includes(query));
    });

    const origin = window.location.origin;
    const postmanUrl = `${origin}/api/v1/postman.json`;
    const openApiUrl = `${origin}/api/v1/openapi.json`;
    const gatewayBaseUrl = `${window.location.protocol}//${window.location.hostname}:8000/api`;

    const getMethodBadgeStyle = (method: string) => {
        switch (method) {
            case 'GET':
                return { bg: '#eff6ff', color: '#1d4ed8', border: '#bfdbfe' };
            case 'POST':
                return { bg: '#ecfdf5', color: '#047857', border: '#a7f3d0' };
            case 'PUT':
                return { bg: '#fffbeb', color: '#b45309', border: '#fde68a' };
            case 'DELETE':
                return { bg: '#fef2f2', color: '#b91c1c', border: '#fecaca' };
            case 'PATCH':
                return { bg: '#f5f3ff', color: '#6d28d9', border: '#ddd6fe' };
            default:
                return { bg: '#f8fafc', color: '#475569', border: '#e2e8f0' };
        }
    };

    if (isLoading) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '300px' }}>
                <div className="spinner"></div>
            </div>
        );
    }

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Settings</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">API Collections</span>
            </div>

            {/* Header */}
            <div className="overview-header" style={{ marginBottom: '8px' }}>
                <div className="overview-title-col">
                    <h1 className="overview-title">API Collections &amp; Documentation</h1>
                    <p className="overview-subtitle">
                        Export your gateway routes into Postman Collections (v2.1) or OpenAPI 3.0 specs, and configure client integration.
                    </p>
                </div>

                <div style={{ display: 'flex', gap: '10px' }}>
                    <button
                        onClick={downloadOpenAPI}
                        style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: '8px',
                            padding: '9px 16px',
                            background: '#ffffff',
                            color: '#1e293b',
                            border: '1px solid #cbd5e1',
                            borderRadius: '8px',
                            fontSize: '13px',
                            fontWeight: 600,
                            cursor: 'pointer',
                            boxShadow: '0 1px 2px rgba(0,0,0,0.05)'
                        }}
                    >
                        <IconDownload width={16} height={16} />
                        Download OpenAPI
                        {selectedIds.size > 0 && selectedIds.size < endpoints.length && ` (${selectedIds.size})`}
                    </button>

                    <button
                        onClick={downloadPostman}
                        style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: '8px',
                            padding: '9px 18px',
                            background: '#2563eb',
                            color: '#ffffff',
                            border: 'none',
                            borderRadius: '8px',
                            fontSize: '13px',
                            fontWeight: 600,
                            cursor: 'pointer',
                            boxShadow: '0 1px 2px rgba(37,99,235,0.2)'
                        }}
                    >
                        <IconDownload width={16} height={16} />
                        Download Postman
                        {selectedIds.size > 0 && selectedIds.size < endpoints.length && ` (${selectedIds.size})`}
                    </button>
                </div>
            </div>

            {/* Quick Export Cards */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: '18px', marginBottom: '28px' }}>
                {/* Postman Collection Card */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '12px' }}>
                        <div style={{ width: '40px', height: '40px', borderRadius: '10px', background: '#fff7ed', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ea580c' }}>
                            <IconFileText width={22} height={22} />
                        </div>
                        <div>
                            <h3 style={{ margin: 0, fontSize: '15px', fontWeight: 700, color: '#0f172a' }}>Postman Collection</h3>
                            <span style={{ fontSize: '12px', color: '#64748b' }}>Format v2.1 Schema</span>
                        </div>
                    </div>
                    <p style={{ fontSize: '13px', color: '#475569', marginBottom: '14px', lineHeight: 1.5 }}>
                        Import directly into Postman Desktop or Web to test requests and configure environments.
                    </p>
                    <div style={{ display: 'flex', alignItems: 'center', background: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '8px', padding: '6px 10px', gap: '8px' }}>
                        <code style={{ fontSize: '12px', color: '#334155', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {postmanUrl}
                        </code>
                        <button
                            onClick={() => copyToClipboard(postmanUrl, 'postmanUrl', 'Postman Collection URL')}
                            style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '4px', color: copiedKey === 'postmanUrl' ? '#16a34a' : '#64748b' }}
                            title="Copy URL"
                        >
                            {copiedKey === 'postmanUrl' ? <IconCheck width={16} height={16} /> : <IconCopy width={16} height={16} />}
                        </button>
                    </div>
                </div>

                {/* OpenAPI Spec Card */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '12px' }}>
                        <div style={{ width: '40px', height: '40px', borderRadius: '10px', background: '#ecfdf5', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#059669' }}>
                            <IconBook width={22} height={22} />
                        </div>
                        <div>
                            <h3 style={{ margin: 0, fontSize: '15px', fontWeight: 700, color: '#0f172a' }}>OpenAPI 3.0 Spec</h3>
                            <span style={{ fontSize: '12px', color: '#64748b' }}>Swagger / OpenAPI Standard</span>
                        </div>
                    </div>
                    <p style={{ fontSize: '13px', color: '#475569', marginBottom: '14px', lineHeight: 1.5 }}>
                        Compatible with Swagger UI, Redoc, SDK generators, and API client tooling.
                    </p>
                    <div style={{ display: 'flex', alignItems: 'center', background: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '8px', padding: '6px 10px', gap: '8px' }}>
                        <code style={{ fontSize: '12px', color: '#334155', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {openApiUrl}
                        </code>
                        <button
                            onClick={() => copyToClipboard(openApiUrl, 'openApiUrl', 'OpenAPI Spec URL')}
                            style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '4px', color: copiedKey === 'openApiUrl' ? '#16a34a' : '#64748b' }}
                            title="Copy URL"
                        >
                            {copiedKey === 'openApiUrl' ? <IconCheck width={16} height={16} /> : <IconCopy width={16} height={16} />}
                        </button>
                    </div>
                </div>

                {/* Gateway Proxy URL Card */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '12px' }}>
                        <div style={{ width: '40px', height: '40px', borderRadius: '10px', background: '#eff6ff', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#2563eb' }}>
                            <IconBolt width={22} height={22} />
                        </div>
                        <div>
                            <h3 style={{ margin: 0, fontSize: '15px', fontWeight: 700, color: '#0f172a' }}>Data Plane Gateway</h3>
                            <span style={{ fontSize: '12px', color: '#64748b' }}>Proxy Port 8000</span>
                        </div>
                    </div>
                    <p style={{ fontSize: '13px', color: '#475569', marginBottom: '14px', lineHeight: 1.5 }}>
                        Target endpoint base URL for all client HTTP requests and proxy routing.
                    </p>
                    <div style={{ display: 'flex', alignItems: 'center', background: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '8px', padding: '6px 10px', gap: '8px' }}>
                        <code style={{ fontSize: '12px', color: '#334155', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {gatewayBaseUrl}
                        </code>
                        <button
                            onClick={() => copyToClipboard(gatewayBaseUrl, 'gatewayBaseUrl', 'Gateway Base URL')}
                            style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '4px', color: copiedKey === 'gatewayBaseUrl' ? '#16a34a' : '#64748b' }}
                            title="Copy URL"
                        >
                            {copiedKey === 'gatewayBaseUrl' ? <IconCheck width={16} height={16} /> : <IconCopy width={16} height={16} />}
                        </button>
                    </div>
                </div>
            </div>

            {/* Published Endpoints Section */}
            <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', overflow: 'hidden', marginBottom: '28px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                <div style={{ padding: '18px 24px', borderBottom: '1px solid #e2e8f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '14px' }}>
                    <div>
                        <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 4px' }}>
                            Published Endpoints in Collection
                        </h2>
                        <span style={{ fontSize: '13px', color: '#64748b' }}>
                            {selectedIds.size > 0
                                ? `${selectedIds.size} of ${endpoints.length} endpoints selected for export`
                                : `All ${endpoints.length} endpoints included in export`}
                        </span>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        {/* Search Input */}
                        <div style={{ position: 'relative', width: '260px' }}>
                            <span style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', color: '#94a3b8', display: 'flex' }}>
                                <IconSearch width={16} height={16} />
                            </span>
                            <input
                                type="text"
                                placeholder="Filter endpoints..."
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                style={{
                                    width: '100%',
                                    padding: '7px 12px 7px 32px',
                                    border: '1px solid #cbd5e1',
                                    borderRadius: '6px',
                                    fontSize: '13px',
                                    outline: 'none'
                                }}
                            />
                        </div>
                    </div>
                </div>

                {filteredEndpoints.length > 0 ? (
                    <div style={{ overflowX: 'auto' }}>
                        <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '13px' }}>
                            <thead>
                                <tr style={{ background: '#f8fafc', borderBottom: '1px solid #e2e8f0', color: '#475569', fontWeight: 600 }}>
                                    <th style={{ width: '48px', padding: '12px 16px', textAlign: 'center' }}>
                                        <input
                                            type="checkbox"
                                            checked={filteredEndpoints.length > 0 && selectedIds.size === filteredEndpoints.length}
                                            onChange={(e) => toggleSelectAll(e.target.checked)}
                                            style={{ cursor: 'pointer', width: '16px', height: '16px' }}
                                        />
                                    </th>
                                    <th style={{ width: '110px', padding: '12px 16px' }}>Method</th>
                                    <th style={{ padding: '12px 16px' }}>Route Path</th>
                                    <th style={{ padding: '12px 16px' }}>Description / Summary</th>
                                    <th style={{ width: '100px', padding: '12px 16px', textAlign: 'right' }}>Action</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredEndpoints.map((ep) => {
                                    const badge = getMethodBadgeStyle(ep.method);
                                    const isSelected = selectedIds.has(ep.id);
                                    const curlSnippet = `curl -X ${ep.method} "${gatewayBaseUrl}${ep.path.startsWith('/') ? ep.path : '/' + ep.path}"`;

                                    return (
                                        <tr
                                            key={ep.id}
                                            style={{
                                                borderBottom: '1px solid #f1f5f9',
                                                background: isSelected ? '#f8faff' : 'transparent',
                                                transition: 'background 0.15s ease'
                                            }}
                                        >
                                            <td style={{ padding: '12px 16px', textAlign: 'center' }}>
                                                <input
                                                    type="checkbox"
                                                    checked={isSelected}
                                                    onChange={() => toggleSelect(ep.id)}
                                                    style={{ cursor: 'pointer', width: '16px', height: '16px' }}
                                                />
                                            </td>
                                            <td style={{ padding: '12px 16px' }}>
                                                <span
                                                    style={{
                                                        display: 'inline-block',
                                                        padding: '3px 8px',
                                                        borderRadius: '6px',
                                                        fontSize: '11.5px',
                                                        fontWeight: 700,
                                                        background: badge.bg,
                                                        color: badge.color,
                                                        border: `1px solid ${badge.border}`
                                                    }}
                                                >
                                                    {ep.method}
                                                </span>
                                            </td>
                                            <td style={{ padding: '12px 16px' }}>
                                                <code style={{ fontSize: '13px', fontWeight: 600, color: '#0f172a', fontFamily: 'monospace' }}>
                                                    {ep.path}
                                                </code>
                                            </td>
                                            <td style={{ padding: '12px 16px', color: '#64748b' }}>
                                                {ep.summary || ep.description || 'Gateway managed endpoint'}
                                            </td>
                                            <td style={{ padding: '12px 16px', textAlign: 'right' }}>
                                                <button
                                                    onClick={() => copyToClipboard(curlSnippet, `curl-${ep.id}`, 'cURL snippet')}
                                                    style={{
                                                        display: 'inline-flex',
                                                        alignItems: 'center',
                                                        gap: '4px',
                                                        padding: '4px 8px',
                                                        background: '#ffffff',
                                                        border: '1px solid #cbd5e1',
                                                        borderRadius: '6px',
                                                        fontSize: '11.5px',
                                                        color: copiedKey === `curl-${ep.id}` ? '#16a34a' : '#475569',
                                                        cursor: 'pointer'
                                                    }}
                                                    title="Copy cURL Command"
                                                >
                                                    {copiedKey === `curl-${ep.id}` ? <IconCheck width={13} height={13} /> : <IconCopy width={13} height={13} />}
                                                    <span>{copiedKey === `curl-${ep.id}` ? 'Copied' : 'cURL'}</span>
                                                </button>
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                ) : (
                    <div style={{ padding: '48px 20px', textAlign: 'center', color: '#64748b' }}>
                        <IconBook width={36} height={36} style={{ margin: '0 auto 12px', color: '#cbd5e1' }} />
                        <h4 style={{ margin: '0 0 6px', fontSize: '15px', fontWeight: 600, color: '#1e293b' }}>
                            {searchQuery ? 'No matching endpoints found' : 'No routes published yet'}
                        </h4>
                        <p style={{ margin: 0, fontSize: '13px' }}>
                            {searchQuery ? 'Try clearing your search query' : 'Create routes in Gateway to populate this API collection.'}
                        </p>
                    </div>
                )}
            </div>

            {/* Authentication & Integration Guide */}
            <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '16px' }}>
                    <div style={{ width: '32px', height: '32px', borderRadius: '8px', background: '#f1f5f9', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#0f172a' }}>
                        <IconKey width={18} height={18} />
                    </div>
                    <div>
                        <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                            Authentication & Client Integration Guide
                        </h2>
                        <span style={{ fontSize: '13px', color: '#64748b' }}>
                            Sample headers and payload requests to integrate external apps with this collection.
                        </span>
                    </div>
                </div>

                {/* Tab Navigation */}
                <div style={{ display: 'flex', gap: '8px', borderBottom: '1px solid #e2e8f0', marginBottom: '16px' }}>
                    <button
                        onClick={() => setActiveAuthTab('bearer')}
                        style={{
                            padding: '8px 14px',
                            fontSize: '13px',
                            fontWeight: 600,
                            background: 'none',
                            border: 'none',
                            borderBottom: activeAuthTab === 'bearer' ? '2px solid #2563eb' : '2px solid transparent',
                            color: activeAuthTab === 'bearer' ? '#2563eb' : '#64748b',
                            cursor: 'pointer'
                        }}
                    >
                        Bearer Token Header (Standard)
                    </button>
                    <button
                        onClick={() => setActiveAuthTab('apiKey')}
                        style={{
                            padding: '8px 14px',
                            fontSize: '13px',
                            fontWeight: 600,
                            background: 'none',
                            border: 'none',
                            borderBottom: activeAuthTab === 'apiKey' ? '2px solid #2563eb' : '2px solid transparent',
                            color: activeAuthTab === 'apiKey' ? '#2563eb' : '#64748b',
                            cursor: 'pointer'
                        }}
                    >
                        X-API-Token Header
                    </button>
                    <button
                        onClick={() => setActiveAuthTab('clientLogin')}
                        style={{
                            padding: '8px 14px',
                            fontSize: '13px',
                            fontWeight: 600,
                            background: 'none',
                            border: 'none',
                            borderBottom: activeAuthTab === 'clientLogin' ? '2px solid #2563eb' : '2px solid transparent',
                            color: activeAuthTab === 'clientLogin' ? '#2563eb' : '#64748b',
                            cursor: 'pointer'
                        }}
                    >
                        Client Auth (POST /api/login)
                    </button>
                    <button
                        onClick={() => setActiveAuthTab('adminLogin')}
                        style={{
                            padding: '8px 14px',
                            fontSize: '13px',
                            fontWeight: 600,
                            background: 'none',
                            border: 'none',
                            borderBottom: activeAuthTab === 'adminLogin' ? '2px solid #2563eb' : '2px solid transparent',
                            color: activeAuthTab === 'adminLogin' ? '#2563eb' : '#64748b',
                            cursor: 'pointer'
                        }}
                    >
                        Admin Auth (POST /auth/login)
                    </button>
                </div>

                {/* Tab Contents */}
                <div>
                    {activeAuthTab === 'bearer' && (
                        <div>
                            <p style={{ fontSize: '13px', color: '#475569', marginBottom: '10px' }}>
                                Send the token in the standard HTTP <code>Authorization</code> header using the Bearer schema:
                            </p>
                            <div style={{ position: 'relative' }}>
                                <pre style={{ background: '#0f172a', color: '#f8fafc', padding: '16px', borderRadius: '8px', fontSize: '12.5px', overflowX: 'auto', margin: 0, fontFamily: 'monospace' }}>
{`GET /api/v1/your-route HTTP/1.1
Host: localhost:8000
Authorization: Bearer <your_access_token>
Content-Type: application/json`}
                                </pre>
                                <button
                                    onClick={() => copyToClipboard('Authorization: Bearer <your_access_token>', 'auth-bearer', 'Bearer Header')}
                                    style={{ position: 'absolute', top: '10px', right: '10px', padding: '5px 9px', background: '#334155', color: '#f8fafc', border: 'none', borderRadius: '6px', fontSize: '11.5px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '4px' }}
                                >
                                    {copiedKey === 'auth-bearer' ? <IconCheck width={14} height={14} /> : <IconCopy width={14} height={14} />}
                                    <span>{copiedKey === 'auth-bearer' ? 'Copied' : 'Copy'}</span>
                                </button>
                            </div>
                        </div>
                    )}

                    {activeAuthTab === 'apiKey' && (
                        <div>
                            <p style={{ fontSize: '13px', color: '#475569', marginBottom: '10px' }}>
                                Alternatively, you can use the custom <code>X-API-Token</code> or <code>X-API-Key</code> header:
                            </p>
                            <div style={{ position: 'relative' }}>
                                <pre style={{ background: '#0f172a', color: '#f8fafc', padding: '16px', borderRadius: '8px', fontSize: '12.5px', overflowX: 'auto', margin: 0, fontFamily: 'monospace' }}>
{`GET /api/v1/your-route HTTP/1.1
Host: localhost:8000
X-API-Token: <your_api_token_or_client_secret>
Content-Type: application/json`}
                                </pre>
                                <button
                                    onClick={() => copyToClipboard('X-API-Token: <your_api_token_or_client_secret>', 'auth-xkey', 'X-API-Token Header')}
                                    style={{ position: 'absolute', top: '10px', right: '10px', padding: '5px 9px', background: '#334155', color: '#f8fafc', border: 'none', borderRadius: '6px', fontSize: '11.5px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '4px' }}
                                >
                                    {copiedKey === 'auth-xkey' ? <IconCheck width={14} height={14} /> : <IconCopy width={14} height={14} />}
                                    <span>{copiedKey === 'auth-xkey' ? 'Copied' : 'Copy'}</span>
                                </button>
                            </div>
                        </div>
                    )}

                    {activeAuthTab === 'clientLogin' && (
                        <div>
                            <p style={{ fontSize: '13px', color: '#475569', marginBottom: '10px' }}>
                                External consumers and API clients can authenticate to obtain an access token:
                            </p>
                            <div style={{ position: 'relative' }}>
                                <pre style={{ background: '#0f172a', color: '#f8fafc', padding: '16px', borderRadius: '8px', fontSize: '12.5px', overflowX: 'auto', margin: 0, fontFamily: 'monospace' }}>
{`POST /api/login HTTP/1.1
Host: localhost:8000
Content-Type: application/json

{
  "username": "my_client_app",
  "password": "client_secret_key"
}

// Response:
{
  "status": true,
  "message": "Success",
  "data": {
    "token": "eyJhbGciOi...",
    "expired": "2026-09-29 17:00:00"
  }
}`}
                                </pre>
                                <button
                                    onClick={() => copyToClipboard(`curl -X POST "${gatewayBaseUrl}/login" -H "Content-Type: application/json" -d '{"username":"client","password":"secret"}'`, 'auth-client-curl', 'cURL Login Command')}
                                    style={{ position: 'absolute', top: '10px', right: '10px', padding: '5px 9px', background: '#334155', color: '#f8fafc', border: 'none', borderRadius: '6px', fontSize: '11.5px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '4px' }}
                                >
                                    {copiedKey === 'auth-client-curl' ? <IconCheck width={14} height={14} /> : <IconCopy width={14} height={14} />}
                                    <span>{copiedKey === 'auth-client-curl' ? 'Copied' : 'Copy'}</span>
                                </button>
                            </div>
                        </div>
                    )}

                    {activeAuthTab === 'adminLogin' && (
                        <div>
                            <p style={{ fontSize: '13px', color: '#475569', marginBottom: '10px' }}>
                                Administrative users authenticate with their registered email address:
                            </p>
                            <div style={{ position: 'relative' }}>
                                <pre style={{ background: '#0f172a', color: '#f8fafc', padding: '16px', borderRadius: '8px', fontSize: '12.5px', overflowX: 'auto', margin: 0, fontFamily: 'monospace' }}>
{`POST /auth/login HTTP/1.1
Host: localhost:8001
Content-Type: application/json

{
  "email": "admin@example.com",
  "password": "YourSecurePassword"
}

// Response:
{
  "token": "eyJhbGciOi...",
  "expires_at": "2026-09-29T17:00:00Z",
  "user": {
    "id": "usr-01",
    "email": "admin@example.com",
    "role": "SUPER_ADMIN"
  }
}`}
                                </pre>
                                <button
                                    onClick={() => copyToClipboard(`curl -X POST "http://localhost:8001/api/v1/auth/login" -H "Content-Type: application/json" -d '{"email":"admin@example.com","password":"..."}'`, 'auth-admin-curl', 'cURL Admin Auth')}
                                    style={{ position: 'absolute', top: '10px', right: '10px', padding: '5px 9px', background: '#334155', color: '#f8fafc', border: 'none', borderRadius: '6px', fontSize: '11.5px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '4px' }}
                                >
                                    {copiedKey === 'auth-admin-curl' ? <IconCheck width={14} height={14} /> : <IconCopy width={14} height={14} />}
                                    <span>{copiedKey === 'auth-admin-curl' ? 'Copied' : 'Copy'}</span>
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
