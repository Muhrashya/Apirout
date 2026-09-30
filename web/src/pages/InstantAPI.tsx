import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { apiClient } from '../api/apiClient';
import { IconDatabase, IconBolt, IconSearch, IconEye } from '../components/Icons';

interface Connection {
    id: string;
    name: string;
    type: string;
}

interface ColumnInfo {
    name: string;
    data_type: string;
    is_nullable: boolean;
    default_value?: string;
    is_primary_key: boolean;
}

interface TableInfo {
    name: string;
    type: string;
    primary_key?: string;
    columns: ColumnInfo[];
}

interface SchemaInfo {
    tables: TableInfo[];
    timestamp: number;
}

export default function InstantAPI() {
    const [connections, setConnections] = useState<Connection[]>([]);
    const [selectedConnId, setSelectedConnId] = useState<string>('');
    const [loading, setLoading] = useState(false);
    const [generating, setGenerating] = useState(false);
    const [schema, setSchema] = useState<SchemaInfo | null>(null);
    const [selectedTable, setSelectedTable] = useState<TableInfo | null>(null);
    const [tableSearch, setTableSearch] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [successMsg, setSuccessMsg] = useState<string | null>(null);

    // Query tester state
    const [filterQuery, setFilterQuery] = useState('');
    const [previewData, setPreviewData] = useState<any[] | null>(null);
    const [previewMeta, setPreviewMeta] = useState<any | null>(null);
    const [queryLoading, setQueryLoading] = useState(false);
    const [copiedSnippet, setCopiedSnippet] = useState<string | null>(null);

    useEffect(() => {
        loadConnections();
    }, []);

    const loadConnections = async () => {
        try {
            const res = await apiClient.get<any>('/connections');
            const connsList: Connection[] = Array.isArray(res)
                ? res
                : (Array.isArray(res?.data) ? res.data : []);

            setConnections(connsList);
            if (connsList.length > 0) {
                setSelectedConnId(connsList[0].id);
                fetchSchema(connsList[0].id);
            }
        } catch (err: any) {
            setError(err?.message || 'Failed to load database connections');
        }
    };

    const fetchSchema = async (connId: string) => {
        if (!connId) return;
        setLoading(true);
        setError(null);
        setSchema(null);
        setSelectedTable(null);
        setPreviewData(null);
        try {
            const res = await apiClient.get<any>(`/connections/${connId}/introspect`);
            const tablesList: TableInfo[] = Array.isArray(res?.tables) ? res.tables : [];
            const schemaData: SchemaInfo = {
                tables: tablesList,
                timestamp: res?.timestamp || Date.now(),
            };
            setSchema(schemaData);
            if (tablesList.length > 0) {
                setSelectedTable(tablesList[0]);
            }
        } catch (err: any) {
            setError(err?.message || 'Failed to introspect database schema');
        } finally {
            setLoading(false);
        }
    };

    const handleConnChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
        const id = e.target.value;
        setSelectedConnId(id);
        fetchSchema(id);
    };

    const handleGenerateAllEndpoints = async () => {
        if (!selectedConnId) return;
        if (!confirm('This will automatically generate Gateway REST routes for all tables in this connection. Proceed?')) {
            return;
        }
        setGenerating(true);
        setError(null);
        setSuccessMsg(null);
        try {
            const res = await apiClient.post<{ message: string }>(`/connections/${selectedConnId}/generate-endpoints`, {});
            setSuccessMsg(res.message || 'REST routes successfully generated and published!');
        } catch (err: any) {
            setError(err?.message || 'Failed to generate endpoints');
        } finally {
            setGenerating(false);
        }
    };

    const runLiveQuery = async () => {
        if (!selectedConnId || !selectedTable) return;
        setQueryLoading(true);
        setError(null);
        try {
            // Target the Data Plane Gateway Instant API on port 8000
            const queryParams = filterQuery ? `?${filterQuery}` : '';
            const gatewayUrl = `http://localhost:8000/api/v1/instant/${selectedConnId}/${selectedTable.name}${queryParams}`;
            const token = localStorage.getItem('token');
            const headers: HeadersInit = {};
            if (token) {
                headers['Authorization'] = `Bearer ${token}`;
            }
            const response = await fetch(gatewayUrl, { headers });
            const json = await response.json();
            if (response.ok) {
                setPreviewData(json.data || []);
                setPreviewMeta(json.meta || null);
            } else {
                setError(json.error || 'Query execution returned an error');
            }
        } catch (err: any) {
            setError(err?.message || 'Failed to contact Gateway on port 8000');
        } finally {
            setQueryLoading(false);
        }
    };

    const copyToClipboard = (text: string, id: string) => {
        navigator.clipboard.writeText(text);
        setCopiedSnippet(id);
        setTimeout(() => setCopiedSnippet(null), 2000);
    };

    const selectedConn = connections.find(c => c.id === selectedConnId);
    const filteredTables = (schema?.tables || []).filter(t =>
        t.name.toLowerCase().includes(tableSearch.toLowerCase())
    );

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb Navigation */}
            <div className="overview-breadcrumb">
                <span>Gateway</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Instant API</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Instant Database API Engine</h1>
                    <p className="overview-subtitle">
                        Instantly expose production-grade, filtered REST CRUD endpoints from PostgreSQL or MySQL with zero code.
                    </p>
                </div>

                <div className="overview-header-actions">
                    {connections.length > 0 && (
                        <select
                            className="select select-filter"
                            style={{ minWidth: '220px' }}
                            value={selectedConnId}
                            onChange={handleConnChange}
                        >
                            {connections.map(c => (
                                <option key={c.id} value={c.id}>
                                    {c.name} ({c.type})
                                </option>
                            ))}
                        </select>
                    )}

                    <button
                        className="btn-header-new-route"
                        onClick={handleGenerateAllEndpoints}
                        disabled={generating || !schema || (schema.tables || []).length === 0}
                        style={{
                            opacity: (generating || !schema || (schema?.tables || []).length === 0) ? 0.6 : 1,
                            cursor: (generating || !schema || (schema?.tables || []).length === 0) ? 'not-allowed' : 'pointer'
                        }}
                    >
                        <IconBolt width={15} height={15} />
                        <span>{generating ? 'Generating...' : 'Auto-Generate Routes'}</span>
                    </button>
                </div>
            </div>

            {/* Alerts */}
            {error && (
                <div style={{
                    padding: '12px 16px',
                    backgroundColor: '#fef2f2',
                    border: '1px solid #fecaca',
                    borderRadius: '8px',
                    color: '#dc2626',
                    fontSize: '13.5px',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center'
                }}>
                    <span>{error}</span>
                    <button
                        onClick={() => setError(null)}
                        style={{ background: 'transparent', border: 'none', color: '#dc2626', fontSize: '18px', cursor: 'pointer' }}
                    >×</button>
                </div>
            )}

            {successMsg && (
                <div style={{
                    padding: '12px 16px',
                    backgroundColor: '#f0fdf4',
                    border: '1px solid #bbf7d0',
                    borderRadius: '8px',
                    color: '#16a34a',
                    fontSize: '13.5px',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center'
                }}>
                    <span>{successMsg}</span>
                    <button
                        onClick={() => setSuccessMsg(null)}
                        style={{ background: 'transparent', border: 'none', color: '#16a34a', fontSize: '18px', cursor: 'pointer' }}
                    >×</button>
                </div>
            )}

            {/* Empty State: No database connections */}
            {connections.length === 0 ? (
                <div style={{
                    backgroundColor: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    padding: '60px 24px',
                    textAlign: 'center',
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                }}>
                    <div style={{
                        width: '56px',
                        height: '56px',
                        borderRadius: '50%',
                        backgroundColor: '#eff6ff',
                        color: '#2563eb',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        margin: '0 auto 16px auto',
                    }}>
                        <IconDatabase width={28} height={28} />
                    </div>
                    <h3 style={{ fontSize: '18px', fontWeight: 600, color: '#0f172a', marginBottom: '8px' }}>
                        No Database Connected Yet
                    </h3>
                    <p style={{ fontSize: '14px', color: '#64748b', maxWidth: '480px', margin: '0 auto 24px auto' }}>
                        Instant API connects directly to your PostgreSQL or MySQL database to discover schemas and generate production REST endpoints in real-time.
                    </p>
                    <Link
                        to="/services"
                        className="btn-header-new-route"
                        style={{ textDecoration: 'none', display: 'inline-flex' }}
                    >
                        + Connect Database Service
                    </Link>
                </div>
            ) : loading ? (
                <div style={{
                    backgroundColor: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    padding: '60px 24px',
                    textAlign: 'center',
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                }}>
                    <div className="spinner" style={{ margin: '0 auto 16px auto' }}></div>
                    <p style={{ fontSize: '14px', color: '#64748b' }}>Introspecting database tables and schemas...</p>
                </div>
            ) : !schema || (schema.tables || []).length === 0 ? (
                <div style={{
                    backgroundColor: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    padding: '60px 24px',
                    textAlign: 'center',
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                }}>
                    <div style={{
                        width: '56px',
                        height: '56px',
                        borderRadius: '50%',
                        backgroundColor: '#f8fafc',
                        color: '#94a3b8',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        margin: '0 auto 16px auto',
                    }}>
                        <IconDatabase width={28} height={28} />
                    </div>
                    <h3 style={{ fontSize: '18px', fontWeight: 600, color: '#0f172a', marginBottom: '8px' }}>
                        No Tables Found
                    </h3>
                    <p style={{ fontSize: '14px', color: '#64748b', maxWidth: '440px', margin: '0 auto 20px auto' }}>
                        Connected to database <strong>{selectedConn?.name}</strong>, but no user tables were found in the public schema.
                    </p>
                    <button
                        className="btn-header-refresh"
                        onClick={() => selectedConnId && fetchSchema(selectedConnId)}
                    >
                        Re-scan Schema
                    </button>
                </div>
            ) : (
                /* Main 2-column Layout */
                <div style={{ display: 'grid', gridTemplateColumns: '280px 1fr', gap: '20px', alignItems: 'start' }}>
                    {/* Left: Introspected Tables List */}
                    <div style={{
                        backgroundColor: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '12px',
                        padding: '16px',
                        boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                    }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                            <span style={{ fontSize: '13px', fontWeight: 600, color: '#0f172a' }}>
                                TABLES ({schema.tables.length})
                            </span>
                            <span style={{
                                fontSize: '11px',
                                fontWeight: 600,
                                textTransform: 'uppercase',
                                padding: '2px 8px',
                                borderRadius: '12px',
                                backgroundColor: '#eff6ff',
                                color: '#2563eb'
                            }}>
                                {selectedConn?.type}
                            </span>
                        </div>

                        {/* Search Table */}
                        <div style={{
                            display: 'flex',
                            alignItems: 'center',
                            backgroundColor: '#f8fafc',
                            border: '1px solid #e2e8f0',
                            borderRadius: '6px',
                            padding: '0 8px',
                            marginBottom: '12px'
                        }}>
                            <IconSearch width={14} height={14} style={{ color: '#94a3b8', marginRight: '6px' }} />
                            <input
                                type="text"
                                placeholder="Filter tables..."
                                value={tableSearch}
                                onChange={e => setTableSearch(e.target.value)}
                                style={{
                                    border: 'none',
                                    background: 'transparent',
                                    fontSize: '12.5px',
                                    padding: '6px 0',
                                    width: '100%',
                                    outline: 'none',
                                    color: '#0f172a'
                                }}
                            />
                        </div>

                        {/* Table Items */}
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', maxHeight: '560px', overflowY: 'auto' }}>
                            {filteredTables.map(t => {
                                const isSelected = selectedTable?.name === t.name;
                                return (
                                    <button
                                        key={t.name}
                                        onClick={() => {
                                            setSelectedTable(t);
                                            setPreviewData(null);
                                            setPreviewMeta(null);
                                        }}
                                        style={{
                                            textAlign: 'left',
                                            padding: '8px 12px',
                                            borderRadius: '6px',
                                            border: 'none',
                                            cursor: 'pointer',
                                            display: 'flex',
                                            justifyContent: 'space-between',
                                            alignItems: 'center',
                                            fontSize: '13px',
                                            backgroundColor: isSelected ? '#2563eb' : 'transparent',
                                            color: isSelected ? '#ffffff' : '#334155',
                                            fontWeight: isSelected ? 600 : 500,
                                            transition: 'all 150ms ease'
                                        }}
                                    >
                                        <span style={{ fontFamily: 'monospace' }}>{t.name}</span>
                                        <span style={{
                                            fontSize: '11px',
                                            opacity: isSelected ? 0.9 : 0.6,
                                            backgroundColor: isSelected ? 'rgba(255,255,255,0.2)' : '#f1f5f9',
                                            padding: '1px 6px',
                                            borderRadius: '4px',
                                            color: isSelected ? '#ffffff' : '#64748b'
                                        }}>
                                            {t.columns.length} cols
                                        </span>
                                    </button>
                                );
                            })}
                        </div>
                    </div>

                    {/* Right: Table Details, Live Query Tester & Snippets */}
                    {selectedTable && (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
                            {/* Table Schema Card */}
                            <div style={{
                                backgroundColor: '#ffffff',
                                border: '1px solid #e2e8f0',
                                borderRadius: '12px',
                                padding: '20px',
                                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                            }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                                    <div>
                                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                            <h2 style={{ fontSize: '18px', fontWeight: 700, fontFamily: 'monospace', color: '#0f172a', margin: 0 }}>
                                                {selectedTable.name}
                                            </h2>
                                            {selectedTable.primary_key && (
                                                <span style={{
                                                    fontSize: '11px',
                                                    fontWeight: 600,
                                                    backgroundColor: '#dcfce7',
                                                    color: '#16a34a',
                                                    padding: '2px 8px',
                                                    borderRadius: '12px'
                                                }}>
                                                    PK: {selectedTable.primary_key}
                                                </span>
                                            )}
                                        </div>
                                        <p style={{ fontSize: '12.5px', color: '#64748b', margin: '4px 0 0 0' }}>
                                            Instant Gateway URL: <code style={{ backgroundColor: '#f1f5f9', padding: '2px 6px', borderRadius: '4px', color: '#0f172a' }}>http://localhost:8000/api/v1/instant/{selectedConnId}/{selectedTable.name}</code>
                                        </p>
                                    </div>
                                </div>

                                {/* Columns Table */}
                                <div style={{ overflowX: 'auto', border: '1px solid #e2e8f0', borderRadius: '8px', marginBottom: '20px' }}>
                                    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                                        <thead>
                                            <tr style={{ textAlign: 'left', backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                                <th style={{ padding: '8px 12px', fontWeight: 600, color: '#475569' }}>Column</th>
                                                <th style={{ padding: '8px 12px', fontWeight: 600, color: '#475569' }}>Data Type</th>
                                                <th style={{ padding: '8px 12px', fontWeight: 600, color: '#475569' }}>Nullable</th>
                                                <th style={{ padding: '8px 12px', fontWeight: 600, color: '#475569' }}>Primary Key</th>
                                                <th style={{ padding: '8px 12px', fontWeight: 600, color: '#475569' }}>Default Value</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {selectedTable.columns.map((col, idx) => (
                                                <tr
                                                    key={col.name}
                                                    style={{
                                                        borderTop: idx > 0 ? '1px solid #f1f5f9' : 'none',
                                                        backgroundColor: idx % 2 === 0 ? '#ffffff' : '#fafafa'
                                                    }}
                                                >
                                                    <td style={{ padding: '8px 12px', fontFamily: 'monospace', fontWeight: 600, color: '#0f172a' }}>{col.name}</td>
                                                    <td style={{ padding: '8px 12px', fontFamily: 'monospace', color: '#64748b' }}>{col.data_type}</td>
                                                    <td style={{ padding: '8px 12px' }}>
                                                        <span style={{
                                                            fontSize: '11px',
                                                            fontWeight: 600,
                                                            padding: '2px 6px',
                                                            borderRadius: '4px',
                                                            backgroundColor: col.is_nullable ? '#f1f5f9' : '#fef2f2',
                                                            color: col.is_nullable ? '#64748b' : '#ef4444'
                                                        }}>
                                                            {col.is_nullable ? 'NULLABLE' : 'REQUIRED'}
                                                        </span>
                                                    </td>
                                                    <td style={{ padding: '8px 12px' }}>
                                                        {col.is_primary_key ? (
                                                            <span style={{ fontSize: '11px', fontWeight: 600, backgroundColor: '#dcfce7', color: '#16a34a', padding: '2px 6px', borderRadius: '4px' }}>
                                                                PRIMARY
                                                            </span>
                                                        ) : (
                                                            <span style={{ color: '#cbd5e1' }}>-</span>
                                                        )}
                                                    </td>
                                                    <td style={{ padding: '8px 12px', fontFamily: 'monospace', fontSize: '12px', color: '#64748b' }}>
                                                        {col.default_value || <span style={{ color: '#cbd5e1' }}>-</span>}
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>

                                {/* Live Query Tester */}
                                <div style={{
                                    backgroundColor: '#f8fafc',
                                    border: '1px solid #e2e8f0',
                                    borderRadius: '8px',
                                    padding: '16px'
                                }}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                                        <h4 style={{ fontSize: '13.5px', fontWeight: 600, color: '#0f172a', margin: 0 }}>
                                            Live Query Tester (Filtering &amp; Pagination)
                                        </h4>
                                        <span style={{ fontSize: '12px', color: '#64748b' }}>
                                            Port 8000 Data Plane
                                        </span>
                                    </div>

                                    <div style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
                                        <div style={{
                                            flex: 1,
                                            display: 'flex',
                                            alignItems: 'center',
                                            backgroundColor: '#ffffff',
                                            border: '1px solid #cbd5e1',
                                            borderRadius: '6px',
                                            padding: '0 10px'
                                        }}>
                                            <span style={{ fontSize: '13px', fontFamily: 'monospace', color: '#94a3b8', marginRight: '6px' }}>?</span>
                                            <input
                                                type="text"
                                                style={{
                                                    width: '100%',
                                                    background: 'transparent',
                                                    border: 'none',
                                                    fontSize: '13px',
                                                    fontFamily: 'monospace',
                                                    outline: 'none',
                                                    padding: '8px 0',
                                                    color: '#0f172a'
                                                }}
                                                placeholder="limit=10&order=id.desc&status=eq.active"
                                                value={filterQuery}
                                                onChange={e => setFilterQuery(e.target.value)}
                                            />
                                        </div>
                                        <button
                                            className="btn-header-refresh"
                                            onClick={runLiveQuery}
                                            disabled={queryLoading}
                                            style={{ backgroundColor: '#2563eb', color: '#ffffff', borderColor: '#2563eb' }}
                                        >
                                            <IconEye width={14} height={14} />
                                            <span>{queryLoading ? 'Executing...' : 'Execute GET'}</span>
                                        </button>
                                    </div>

                                    {previewMeta && (
                                        <div style={{ fontSize: '12px', color: '#64748b', marginBottom: '8px', display: 'flex', gap: '16px' }}>
                                            <span>Total: <strong>{previewMeta.total}</strong></span>
                                            <span>Limit: <strong>{previewMeta.limit}</strong></span>
                                            <span>Offset: <strong>{previewMeta.offset}</strong></span>
                                            <span>Returned: <strong>{previewMeta.returned}</strong></span>
                                        </div>
                                    )}

                                    {previewData && (
                                        <pre style={{
                                            backgroundColor: '#0f172a',
                                            color: '#34d399',
                                            padding: '12px',
                                            borderRadius: '6px',
                                            fontSize: '12px',
                                            overflowX: 'auto',
                                            maxHeight: '260px',
                                            fontFamily: 'monospace',
                                            margin: 0
                                        }}>
                                            {JSON.stringify(previewData, null, 2)}
                                        </pre>
                                    )}
                                </div>
                            </div>

                            {/* Cheatsheet Card */}
                            <div style={{
                                backgroundColor: '#ffffff',
                                border: '1px solid #e2e8f0',
                                borderRadius: '12px',
                                padding: '20px',
                                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                            }}>
                                <h4 style={{ fontSize: '14px', fontWeight: 600, color: '#0f172a', marginBottom: '12px' }}>
                                    Instant REST Endpoints Cheatsheet
                                </h4>
                                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', fontSize: '12.5px', fontFamily: 'monospace' }}>
                                    {[
                                        { id: 'list', method: 'GET', color: '#16a34a', bg: '#dcfce7', path: `/api/v1/instant/${selectedConnId}/${selectedTable.name}?limit=20&offset=0`, desc: 'List rows' },
                                        { id: 'get', method: 'GET', color: '#2563eb', bg: '#dbeafe', path: `/api/v1/instant/${selectedConnId}/${selectedTable.name}/:id`, desc: 'Get row by ID' },
                                        { id: 'post', method: 'POST', color: '#d97706', bg: '#fef3c7', path: `/api/v1/instant/${selectedConnId}/${selectedTable.name}`, desc: 'Insert new row' },
                                        { id: 'put', method: 'PUT', color: '#7c3aed', bg: '#ede9fe', path: `/api/v1/instant/${selectedConnId}/${selectedTable.name}/:id`, desc: 'Update row' },
                                        { id: 'del', method: 'DELETE', color: '#dc2626', bg: '#fee2e2', path: `/api/v1/instant/${selectedConnId}/${selectedTable.name}/:id`, desc: 'Delete row' },
                                    ].map(item => (
                                        <div
                                            key={item.id}
                                            style={{
                                                backgroundColor: '#f8fafc',
                                                border: '1px solid #e2e8f0',
                                                padding: '8px 12px',
                                                borderRadius: '6px',
                                                display: 'flex',
                                                justifyContent: 'space-between',
                                                alignItems: 'center'
                                            }}
                                        >
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                                <span style={{
                                                    fontSize: '11px',
                                                    fontWeight: 700,
                                                    padding: '2px 6px',
                                                    borderRadius: '4px',
                                                    backgroundColor: item.bg,
                                                    color: item.color
                                                }}>
                                                    {item.method}
                                                </span>
                                                <span style={{ color: '#0f172a' }}>{item.path}</span>
                                            </div>
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                                <span style={{ color: '#64748b', fontSize: '11.5px', fontFamily: 'sans-serif' }}>{item.desc}</span>
                                                <button
                                                    onClick={() => copyToClipboard(`curl -X ${item.method} "http://localhost:8000${item.path}"`, item.id)}
                                                    style={{
                                                        background: 'transparent',
                                                        border: '1px solid #cbd5e1',
                                                        borderRadius: '4px',
                                                        padding: '2px 8px',
                                                        fontSize: '11px',
                                                        color: '#475569',
                                                        cursor: 'pointer'
                                                    }}
                                                >
                                                    {copiedSnippet === item.id ? 'Copied!' : 'Copy cURL'}
                                                </button>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
