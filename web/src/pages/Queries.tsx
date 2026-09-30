import React, { useState, useEffect, useRef } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { gatewayApi, QueryItem, Connection } from '../api/gatewayApi';
import { useToast } from '../components/Toast';
import { Modal } from '../components/Modal';
import { useAuth } from '../auth/AuthProvider';
import {
    IconQuery,
    IconDatabase,
    IconPlay,
    IconPlus,
    IconRefresh,
    IconSearch,
    IconTrash,
    IconPencil,
    IconCheck,
    IconCopy
} from '../components/Icons';

export default function QueriesPage() {
    const navigate = useNavigate();
    const toast = useToast();
    const { canManageEndpoints } = useAuth();
    const canManage = canManageEndpoints();

    // Data State
    const [queries, setQueries] = useState<QueryItem[]>([]);
    const [connections, setConnections] = useState<Connection[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const [connectionFilter, setConnectionFilter] = useState('all');

    // Interactive Runner State
    const [activeConnectionId, setActiveConnectionId] = useState<string>('');
    const [activeSql, setActiveSql] = useState<string>('SELECT * FROM users LIMIT 10;');
    const [previewLimit, setPreviewLimit] = useState<number>(25);
    const [isRunningPreview, setIsRunningPreview] = useState(false);
    const [previewResults, setPreviewResults] = useState<any[] | null>(null);
    const [previewError, setPreviewError] = useState<string | null>(null);
    const [previewExecutionTime, setPreviewExecutionTime] = useState<number | null>(null);
    const editorRef = useRef<HTMLDivElement>(null);

    // Modal State (Create / Edit Saved Query)
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [editingQuery, setEditingQuery] = useState<QueryItem | null>(null);
    const [queryForm, setQueryForm] = useState({
        name: '',
        connection_id: '',
        sql_text: '',
        default_limit: 100,
        description: ''
    });
    const [isSaving, setIsSaving] = useState(false);
    const [copiedId, setCopiedId] = useState<string | null>(null);

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setIsLoading(true);
        try {
            const [queriesRes, connsRes] = await Promise.all([
                gatewayApi.getQueries().catch(() => ({ data: [] })),
                gatewayApi.getConnections().catch(() => ({ data: [] }))
            ]);

            const loadedQueries = (queriesRes as any)?.data || (Array.isArray(queriesRes) ? queriesRes : []);
            const loadedConns = (connsRes as any)?.data || (Array.isArray(connsRes) ? connsRes : []);

            setQueries(loadedQueries);
            setConnections(loadedConns);

            if (loadedConns.length > 0 && !activeConnectionId) {
                setActiveConnectionId(loadedConns[0].id);
            }
        } catch (err: any) {
            toast.error(err.message || 'Failed to load queries data');
        } finally {
            setIsLoading(false);
        }
    };

    // Execute Live SQL Preview
    const handleRunPreview = async () => {
        if (!activeConnectionId) {
            toast.error('Please select a Database Connection to execute against');
            return;
        }
        if (!activeSql.trim()) {
            toast.error('Please enter a SQL statement to execute');
            return;
        }

        setIsRunningPreview(true);
        setPreviewError(null);
        setPreviewResults(null);
        setPreviewExecutionTime(null);

        const startTime = performance.now();
        try {
            const res = await gatewayApi.previewQuery({
                connection_id: activeConnectionId,
                sql: activeSql.trim(),
                limit: previewLimit,
                offset: 0
            });

            const elapsed = Math.round(performance.now() - startTime);
            setPreviewExecutionTime(elapsed);

            const rows = (res as any)?.data || (Array.isArray(res) ? res : []);
            setPreviewResults(rows);
            toast.success(`Query executed successfully (${elapsed}ms)`);
        } catch (err: any) {
            const elapsed = Math.round(performance.now() - startTime);
            setPreviewExecutionTime(elapsed);
            setPreviewError(err.message || 'Failed to execute query');
            toast.error(err.message || 'Query execution failed');
        } finally {
            setIsRunningPreview(false);
        }
    };

    // Open Modal for New Query
    const handleOpenCreateModal = () => {
        setEditingQuery(null);
        setQueryForm({
            name: '',
            connection_id: activeConnectionId || (connections[0]?.id || ''),
            sql_text: activeSql || 'SELECT * FROM users LIMIT 10;',
            default_limit: 100,
            description: ''
        });
        setIsModalOpen(true);
    };

    // Open Modal for Edit
    const handleOpenEditModal = (q: QueryItem) => {
        setEditingQuery(q);
        setQueryForm({
            name: q.name || '',
            connection_id: q.connection_id || '',
            sql_text: q.sql_text || '',
            default_limit: (q as any).default_limit || 100,
            description: q.description || ''
        });
        setIsModalOpen(true);
    };

    // Load saved query into top interactive runner
    const handleLoadIntoEditor = (q: QueryItem) => {
        setActiveConnectionId(q.connection_id);
        setActiveSql(q.sql_text);
        if (editorRef.current) {
            editorRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }
        toast.info(`Loaded "${q.name}" into SQL Editor`);
    };

    // Save or Update Query
    const handleSaveQuery = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!queryForm.name.trim()) {
            toast.error('Query Name is required');
            return;
        }
        if (!queryForm.connection_id) {
            toast.error('Database Connection is required');
            return;
        }
        if (!queryForm.sql_text.trim()) {
            toast.error('SQL Statement is required');
            return;
        }

        setIsSaving(true);
        try {
            const payload = {
                name: queryForm.name.trim(),
                connection_id: queryForm.connection_id,
                sql_text: queryForm.sql_text.trim(),
                default_limit: Number(queryForm.default_limit) || 100,
                description: queryForm.description.trim() || undefined
            };

            if (editingQuery) {
                await gatewayApi.updateQuery(editingQuery.id, payload);
                toast.success(`Query "${payload.name}" updated successfully`);
            } else {
                await gatewayApi.createQuery(payload);
                toast.success(`Query "${payload.name}" created successfully`);
            }
            setIsModalOpen(false);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to save query');
        } finally {
            setIsSaving(false);
        }
    };

    // Delete Query
    const handleDeleteQuery = async (q: QueryItem) => {
        if (!window.confirm(`Are you sure you want to delete query "${q.name}"? Routes bound to this query may stop functioning.`)) {
            return;
        }
        try {
            await gatewayApi.deleteQuery(q.id);
            toast.success(`Query "${q.name}" deleted`);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to delete query');
        }
    };

    // Copy SQL text
    const handleCopySql = (id: string, text: string) => {
        navigator.clipboard.writeText(text);
        setCopiedId(id);
        setTimeout(() => setCopiedId(null), 2000);
        toast.info('SQL copied to clipboard');
    };

    // Filter queries list
    const filteredQueries = queries.filter(q => {
        const matchesSearch = !searchTerm ||
            q.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            q.sql_text.toLowerCase().includes(searchTerm.toLowerCase()) ||
            (q.description && q.description.toLowerCase().includes(searchTerm.toLowerCase()));
        const matchesConn = connectionFilter === 'all' || q.connection_id === connectionFilter;
        return matchesSearch && matchesConn;
    });

    const getConnectionName = (connId: string) => {
        const found = connections.find(c => c.id === connId);
        return found ? `${found.name} (${found.db_type})` : connId || 'Default Connection';
    };

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Gateway</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Queries</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">SQL Queries Engine</h1>
                    <p className="overview-subtitle">
                        Execute live SQL against connected databases and expose parameterized queries as high-performance Gateway routes.
                    </p>
                </div>

                <div className="overview-header-actions">
                    <button
                        onClick={loadData}
                        className="btn-header-refresh"
                        title="Reload queries and connections"
                    >
                        <IconRefresh width={14} height={14} />
                        <span>Refresh</span>
                    </button>
                    {canManage && (
                        <button
                            onClick={handleOpenCreateModal}
                            className="btn-header-new-route"
                        >
                            <IconPlus width={15} height={15} />
                            <span>New Query</span>
                        </button>
                    )}
                </div>
            </div>

            {/* Summary Metrics */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '16px', marginBottom: '24px' }}>
                <div style={{ backgroundColor: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '16px 20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ fontSize: '12px', fontWeight: 600, color: '#64748b', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                        Saved Queries
                    </div>
                    <div style={{ fontSize: '26px', fontWeight: 700, color: '#0f172a', marginTop: '6px' }}>
                        {queries.length}
                    </div>
                    <div style={{ fontSize: '12px', color: '#64748b', marginTop: '4px' }}>
                        Ready for gateway binding
                    </div>
                </div>

                <div style={{ backgroundColor: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '16px 20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ fontSize: '12px', fontWeight: 600, color: '#64748b', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                        Target Databases
                    </div>
                    <div style={{ fontSize: '26px', fontWeight: 700, color: '#2563eb', marginTop: '6px' }}>
                        {connections.length}
                    </div>
                    <div style={{ fontSize: '12px', color: '#64748b', marginTop: '4px' }}>
                        <Link to="/settings/connections" style={{ color: '#2563eb', textDecoration: 'none', fontWeight: 500 }}>
                            Manage connections →
                        </Link>
                    </div>
                </div>

                <div style={{ backgroundColor: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '16px 20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ fontSize: '12px', fontWeight: 600, color: '#64748b', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
                        Parameterized
                    </div>
                    <div style={{ fontSize: '26px', fontWeight: 700, color: '#059669', marginTop: '6px' }}>
                        {queries.filter(q => q.sql_text.includes('{{') || q.sql_text.includes('$') || q.sql_text.includes(':')).length}
                    </div>
                    <div style={{ fontSize: '12px', color: '#64748b', marginTop: '4px' }}>
                        Dynamic route placeholders
                    </div>
                </div>
            </div>

            {/* INTERACTIVE SQL EDITOR & RUNNER */}
            <div ref={editorRef} style={{
                backgroundColor: '#ffffff',
                border: '1px solid #e2e8f0',
                borderRadius: '12px',
                marginBottom: '28px',
                boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
                overflow: 'hidden'
            }}>
                <div style={{
                    padding: '16px 20px',
                    borderBottom: '1px solid #f1f5f9',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    flexWrap: 'wrap',
                    gap: '12px',
                    backgroundColor: '#fafbfc'
                }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <span style={{
                            width: '28px',
                            height: '28px',
                            borderRadius: '6px',
                            backgroundColor: '#eff6ff',
                            color: '#2563eb',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center'
                        }}>
                            <IconPlay width={14} height={14} />
                        </span>
                        <div>
                            <h2 style={{ fontSize: '15px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                                Interactive SQL Workbench
                            </h2>
                            <span style={{ fontSize: '12px', color: '#64748b' }}>
                                Test query execution against database connections in real-time
                            </span>
                        </div>
                    </div>

                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                            <label style={{ fontSize: '12.5px', fontWeight: 600, color: '#475569' }}>Database:</label>
                            <select
                                className="select select-filter"
                                style={{ width: '220px' }}
                                value={activeConnectionId}
                                onChange={e => setActiveConnectionId(e.target.value)}
                            >
                                {connections.length === 0 && (
                                    <option value="">No database connections available</option>
                                )}
                                {connections.map(c => (
                                    <option key={c.id} value={c.id}>
                                        {c.name} ({c.db_type})
                                    </option>
                                ))}
                            </select>
                        </div>

                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                            <label style={{ fontSize: '12.5px', fontWeight: 600, color: '#475569' }}>Limit:</label>
                            <select
                                className="select select-filter"
                                style={{ width: '85px', minWidth: '85px' }}
                                value={previewLimit}
                                onChange={e => setPreviewLimit(Number(e.target.value))}
                            >
                                <option value={10}>10</option>
                                <option value={25}>25</option>
                                <option value={50}>50</option>
                                <option value={100}>100</option>
                            </select>
                        </div>

                        <button
                            onClick={handleRunPreview}
                            disabled={isRunningPreview || !activeConnectionId}
                            style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: '6px',
                                backgroundColor: isRunningPreview ? '#93c5fd' : '#2563eb',
                                color: '#ffffff',
                                border: 'none',
                                borderRadius: '8px',
                                padding: '8px 16px',
                                fontSize: '13px',
                                fontWeight: 600,
                                cursor: isRunningPreview ? 'not-allowed' : 'pointer',
                                transition: 'background-color 150ms ease'
                            }}
                        >
                            <IconPlay width={14} height={14} />
                            <span>{isRunningPreview ? 'Executing...' : 'Run Query'}</span>
                        </button>

                        {canManage && (
                            <button
                                onClick={handleOpenCreateModal}
                                style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '6px',
                                    backgroundColor: '#f1f5f9',
                                    color: '#0f172a',
                                    border: '1px solid #cbd5e1',
                                    borderRadius: '8px',
                                    padding: '8px 14px',
                                    fontSize: '13px',
                                    fontWeight: 600,
                                    cursor: 'pointer'
                                }}
                            >
                                <IconPlus width={13} height={13} />
                                <span>Save As Query</span>
                            </button>
                        )}
                    </div>
                </div>

                {/* SQL Textarea */}
                <div style={{ padding: '16px 20px', backgroundColor: '#0f172a' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                        <span style={{ fontSize: '11.5px', color: '#94a3b8', fontFamily: 'monospace' }}>
                            SQL EDITOR &bull; Supports SELECT queries &bull; Use {'{{param}}'} for dynamic route variables
                        </span>
                        <button
                            onClick={() => handleCopySql('workbench', activeSql)}
                            style={{
                                background: 'transparent',
                                border: 'none',
                                color: '#94a3b8',
                                cursor: 'pointer',
                                fontSize: '11.5px',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '4px'
                            }}
                        >
                            {copiedId === 'workbench' ? <IconCheck width={12} height={12} style={{ color: '#10b981' }} /> : <IconCopy width={12} height={12} />}
                            <span>{copiedId === 'workbench' ? 'Copied' : 'Copy SQL'}</span>
                        </button>
                    </div>
                    <textarea
                        value={activeSql}
                        onChange={e => setActiveSql(e.target.value)}
                        placeholder="SELECT * FROM users WHERE status = 'active' LIMIT 10;"
                        rows={5}
                        style={{
                            width: '100%',
                            backgroundColor: '#1e293b',
                            color: '#f8fafc',
                            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                            fontSize: '13.5px',
                            lineHeight: '1.5',
                            border: '1px solid #334155',
                            borderRadius: '8px',
                            padding: '12px 14px',
                            resize: 'vertical',
                            boxSizing: 'border-box',
                            outline: 'none'
                        }}
                    />
                </div>

                {/* Results Panel */}
                <div style={{ padding: '16px 20px', borderTop: '1px solid #e2e8f0' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                            <span style={{ fontSize: '13px', fontWeight: 600, color: '#0f172a' }}>Execution Output</span>
                            {previewExecutionTime !== null && (
                                <span style={{
                                    fontSize: '11.5px',
                                    padding: '2px 8px',
                                    borderRadius: '12px',
                                    backgroundColor: previewError ? '#fef2f2' : '#ecfdf5',
                                    color: previewError ? '#dc2626' : '#059669',
                                    fontWeight: 600
                                }}>
                                    {previewError ? 'Failed' : `Returned ${previewResults?.length || 0} rows`} ({previewExecutionTime} ms)
                                </span>
                            )}
                        </div>
                    </div>

                    {isRunningPreview ? (
                        <div style={{ padding: '40px', textAlign: 'center', color: '#64748b' }}>
                            <div className="spinner" style={{ margin: '0 auto 12px auto' }}></div>
                            <div style={{ fontSize: '13px' }}>Executing query on {getConnectionName(activeConnectionId)}...</div>
                        </div>
                    ) : previewError ? (
                        <div style={{
                            padding: '14px 18px',
                            backgroundColor: '#fef2f2',
                            border: '1px solid #fecaca',
                            borderRadius: '8px',
                            color: '#991b1b',
                            fontSize: '13px',
                            fontFamily: 'monospace',
                            whiteSpace: 'pre-wrap'
                        }}>
                            <strong>Query Error:</strong> {previewError}
                        </div>
                    ) : previewResults ? (
                        previewResults.length === 0 ? (
                            <div style={{ padding: '30px', textAlign: 'center', color: '#64748b', fontSize: '13px', backgroundColor: '#f8fafc', borderRadius: '8px' }}>
                                Query executed successfully, but returned 0 rows.
                            </div>
                        ) : (
                            <div style={{ overflowX: 'auto', maxHeight: '380px', border: '1px solid #e2e8f0', borderRadius: '8px' }}>
                                <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '12.5px' }}>
                                    <thead style={{ position: 'sticky', top: 0, backgroundColor: '#f8fafc', zIndex: 1 }}>
                                        <tr style={{ borderBottom: '1px solid #e2e8f0' }}>
                                            {Object.keys(previewResults[0]).map((col, idx) => (
                                                <th key={idx} style={{ padding: '10px 14px', fontWeight: 600, color: '#334155', whiteSpace: 'nowrap' }}>
                                                    {col}
                                                </th>
                                            ))}
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {previewResults.map((row, rIdx) => (
                                            <tr key={rIdx} style={{ borderTop: rIdx > 0 ? '1px solid #f1f5f9' : 'none', backgroundColor: rIdx % 2 === 0 ? '#ffffff' : '#fafafa' }}>
                                                {Object.keys(previewResults[0]).map((col, cIdx) => {
                                                    const val = row[col];
                                                    const displayVal = val === null || val === undefined
                                                        ? '<null>'
                                                        : typeof val === 'object'
                                                            ? JSON.stringify(val)
                                                            : String(val);

                                                    return (
                                                        <td key={cIdx} style={{
                                                            padding: '9px 14px',
                                                            color: val === null ? '#94a3b8' : '#1e293b',
                                                            fontFamily: typeof val === 'number' || val === null ? 'monospace' : 'inherit',
                                                            whiteSpace: 'nowrap',
                                                            maxWidth: '300px',
                                                            overflow: 'hidden',
                                                            textOverflow: 'ellipsis'
                                                        }}>
                                                            {displayVal}
                                                        </td>
                                                    );
                                                })}
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        )
                    ) : (
                        <div style={{ padding: '30px', textAlign: 'center', color: '#94a3b8', fontSize: '13px', backgroundColor: '#f8fafc', borderRadius: '8px' }}>
                            Click &ldquo;Run Query&rdquo; above to execute and preview data rows.
                        </div>
                    )}
                </div>
            </div>

            {/* SAVED QUERIES REPOSITORY */}
            <div style={{
                backgroundColor: '#ffffff',
                border: '1px solid #e2e8f0',
                borderRadius: '12px',
                overflow: 'hidden',
                boxShadow: '0 1px 3px rgba(0,0,0,0.04)'
            }}>
                {/* Search & Filter Bar */}
                <div style={{
                    padding: '16px 20px',
                    borderBottom: '1px solid #e2e8f0',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    flexWrap: 'wrap',
                    gap: '12px',
                    backgroundColor: '#fafbfc'
                }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flex: '1 1 360px', flexWrap: 'wrap' }}>
                        <div style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '8px',
                            backgroundColor: '#ffffff',
                            border: '1px solid #cbd5e1',
                            borderRadius: '8px',
                            padding: '6px 12px',
                            flex: '1 1 240px'
                        }}>
                            <IconSearch width={15} height={15} style={{ color: '#94a3b8' }} />
                            <input
                                type="text"
                                placeholder="Search queries by name, SQL, or description..."
                                value={searchTerm}
                                onChange={e => setSearchTerm(e.target.value)}
                                style={{
                                    border: 'none',
                                    outline: 'none',
                                    width: '100%',
                                    fontSize: '13px',
                                    color: '#0f172a',
                                    backgroundColor: 'transparent'
                                }}
                            />
                        </div>

                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                            <span style={{ fontSize: '12.5px', color: '#64748b' }}>Filter DB:</span>
                            <select
                                className="select select-filter"
                                style={{ width: '180px' }}
                                value={connectionFilter}
                                onChange={e => setConnectionFilter(e.target.value)}
                            >
                                <option value="all">All Databases ({connections.length})</option>
                                {connections.map(c => (
                                    <option key={c.id} value={c.id}>{c.name}</option>
                                ))}
                            </select>
                        </div>
                    </div>

                    <div style={{ fontSize: '13px', color: '#64748b' }}>
                        Showing <strong>{filteredQueries.length}</strong> of {queries.length} queries
                    </div>
                </div>

                {/* Queries Table */}
                {isLoading ? (
                    <div style={{ padding: '60px', textAlign: 'center' }}>
                        <div className="spinner" style={{ margin: '0 auto 12px auto' }}></div>
                        <div style={{ fontSize: '13px', color: '#64748b' }}>Loading saved queries...</div>
                    </div>
                ) : filteredQueries.length === 0 ? (
                    <div style={{ padding: '60px 24px', textAlign: 'center' }}>
                        <div style={{
                            width: '48px',
                            height: '48px',
                            borderRadius: '50%',
                            backgroundColor: '#eff6ff',
                            color: '#2563eb',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            margin: '0 auto 16px auto'
                        }}>
                            <IconQuery width={24} height={24} />
                        </div>
                        <h3 style={{ fontSize: '16px', fontWeight: 600, color: '#0f172a', marginBottom: '6px' }}>
                            No Queries Found
                        </h3>
                        <p style={{ fontSize: '13.5px', color: '#64748b', maxWidth: '420px', margin: '0 auto 20px auto' }}>
                            {searchTerm || connectionFilter !== 'all'
                                ? 'No queries match your current filter criteria.'
                                : 'Save your first SQL query using the workbench above or click New Query.'}
                        </p>
                        {canManage && (
                            <button onClick={handleOpenCreateModal} className="btn-header-new-route">
                                <IconPlus width={14} height={14} />
                                <span>Create Query</span>
                            </button>
                        )}
                    </div>
                ) : (
                    <div style={{ overflowX: 'auto' }}>
                        <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '13.5px' }}>
                            <thead>
                                <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Query Name</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Target Database</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>SQL Statement</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Default Limit</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569', textAlign: 'right' }}>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredQueries.map((q, idx) => {
                                    const connName = getConnectionName(q.connection_id);
                                    const isParameterized = q.sql_text.includes('{{') || q.sql_text.includes('$') || q.sql_text.includes(':');

                                    return (
                                        <tr
                                            key={q.id}
                                            style={{
                                                borderTop: idx > 0 ? '1px solid #f1f5f9' : 'none',
                                                backgroundColor: idx % 2 === 0 ? '#ffffff' : '#fafafa'
                                            }}
                                        >
                                            <td style={{ padding: '14px 20px', verticalAlign: 'top' }}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                                    <span style={{ fontWeight: 600, color: '#0f172a' }}>{q.name}</span>
                                                    {isParameterized && (
                                                        <span style={{
                                                            fontSize: '10.5px',
                                                            backgroundColor: '#ecfdf5',
                                                            color: '#059669',
                                                            fontWeight: 600,
                                                            padding: '2px 6px',
                                                            borderRadius: '4px'
                                                        }}>
                                                            Parameterized
                                                        </span>
                                                    )}
                                                </div>
                                                {q.description && (
                                                    <div style={{ fontSize: '12px', color: '#64748b', marginTop: '3px' }}>
                                                        {q.description}
                                                    </div>
                                                )}
                                            </td>

                                            <td style={{ padding: '14px 20px', verticalAlign: 'top' }}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                    <IconDatabase width={14} height={14} style={{ color: '#64748b' }} />
                                                    <span style={{ fontSize: '13px', color: '#334155', fontWeight: 500 }}>
                                                        {connName}
                                                    </span>
                                                </div>
                                            </td>

                                            <td style={{ padding: '14px 20px', verticalAlign: 'top', maxWidth: '380px' }}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                    <code style={{
                                                        fontFamily: 'monospace',
                                                        fontSize: '12px',
                                                        backgroundColor: '#f1f5f9',
                                                        padding: '4px 8px',
                                                        borderRadius: '4px',
                                                        color: '#1e293b',
                                                        display: 'inline-block',
                                                        maxWidth: '340px',
                                                        overflow: 'hidden',
                                                        textOverflow: 'ellipsis',
                                                        whiteSpace: 'nowrap'
                                                    }}>
                                                        {q.sql_text}
                                                    </code>
                                                    <button
                                                        onClick={() => handleCopySql(q.id, q.sql_text)}
                                                        title="Copy SQL"
                                                        style={{
                                                            background: 'none',
                                                            border: 'none',
                                                            cursor: 'pointer',
                                                            color: '#94a3b8',
                                                            padding: '2px'
                                                        }}
                                                    >
                                                        {copiedId === q.id ? <IconCheck width={13} height={13} style={{ color: '#10b981' }} /> : <IconCopy width={13} height={13} />}
                                                    </button>
                                                </div>
                                            </td>

                                            <td style={{ padding: '14px 20px', verticalAlign: 'top', color: '#64748b', fontSize: '13px' }}>
                                                {(q as any).default_limit || 100} rows
                                            </td>

                                            <td style={{ padding: '14px 20px', verticalAlign: 'top', textAlign: 'right' }}>
                                                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: '8px' }}>
                                                    <button
                                                        onClick={() => handleLoadIntoEditor(q)}
                                                        className="btn-header-refresh"
                                                        title="Load into interactive workbench to test"
                                                        style={{ fontSize: '12px', padding: '4px 8px', display: 'flex', alignItems: 'center', gap: '4px' }}
                                                    >
                                                        <IconPlay width={11} height={11} />
                                                        <span>Test</span>
                                                    </button>

                                                    <button
                                                        onClick={() => navigate(`/routes/new?query_id=${q.id}`)}
                                                        className="btn-header-new-route"
                                                        title="Expose this query as a Gateway Route"
                                                        style={{ fontSize: '12px', padding: '4px 8px', display: 'flex', alignItems: 'center', gap: '4px' }}
                                                    >
                                                        <span>+ Create Route</span>
                                                    </button>

                                                    {canManage && (
                                                        <>
                                                            <button
                                                                onClick={() => handleOpenEditModal(q)}
                                                                title="Edit Query"
                                                                style={{
                                                                    background: 'none',
                                                                    border: '1px solid #e2e8f0',
                                                                    borderRadius: '6px',
                                                                    padding: '5px',
                                                                    cursor: 'pointer',
                                                                    color: '#475569'
                                                                }}
                                                            >
                                                                <IconPencil width={13} height={13} />
                                                            </button>

                                                            <button
                                                                onClick={() => handleDeleteQuery(q)}
                                                                title="Delete Query"
                                                                style={{
                                                                    background: 'none',
                                                                    border: '1px solid #fee2e2',
                                                                    borderRadius: '6px',
                                                                    padding: '5px',
                                                                    cursor: 'pointer',
                                                                    color: '#ef4444'
                                                                }}
                                                            >
                                                                <IconTrash width={13} height={13} />
                                                            </button>
                                                        </>
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

            {/* CREATE / EDIT QUERY MODAL */}
            <Modal
                isOpen={isModalOpen}
                onClose={() => setIsModalOpen(false)}
                title={editingQuery ? `Edit Query: ${editingQuery.name}` : 'Create Saved SQL Query'}
                size="lg"
            >
                <form onSubmit={handleSaveQuery}>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                        <div>
                            <label className="label">Query Name *</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="e.g. get_active_users or order_summary_by_customer"
                                required
                                value={queryForm.name}
                                onChange={e => setQueryForm(prev => ({ ...prev, name: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                A unique identifier name for this query.
                            </span>
                        </div>

                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 140px', gap: '14px' }}>
                            <div>
                                <label className="label">Target Database Connection *</label>
                                <select
                                    className="select"
                                    required
                                    value={queryForm.connection_id}
                                    onChange={e => setQueryForm(prev => ({ ...prev, connection_id: e.target.value }))}
                                >
                                    <option value="">-- Select Connection --</option>
                                    {connections.map(c => (
                                        <option key={c.id} value={c.id}>
                                            {c.name} ({c.db_type})
                                        </option>
                                    ))}
                                </select>
                            </div>

                            <div>
                                <label className="label">Default Limit</label>
                                <input
                                    type="number"
                                    className="input"
                                    min={1}
                                    max={10000}
                                    value={queryForm.default_limit}
                                    onChange={e => setQueryForm(prev => ({ ...prev, default_limit: parseInt(e.target.value) || 100 }))}
                                />
                            </div>
                        </div>

                        <div>
                            <label className="label">SQL Statement *</label>
                            <textarea
                                className="input"
                                rows={6}
                                required
                                placeholder="SELECT id, name, email, created_at FROM users WHERE status = 'active' ORDER BY id DESC"
                                value={queryForm.sql_text}
                                onChange={e => setQueryForm(prev => ({ ...prev, sql_text: e.target.value }))}
                                style={{
                                    fontFamily: 'monospace',
                                    fontSize: '13px',
                                    lineHeight: '1.5',
                                    resize: 'vertical'
                                }}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Tip: Parameterized queries can extract values directly from route query/body parameters using <code>{'{{param_name}}'}</code>.
                            </span>
                        </div>

                        <div>
                            <label className="label">Description / Documentation</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="Brief description of what this query fetches or computes"
                                value={queryForm.description}
                                onChange={e => setQueryForm(prev => ({ ...prev, description: e.target.value }))}
                            />
                        </div>

                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '12px', borderTop: '1px solid #f1f5f9', paddingTop: '16px' }}>
                            <button
                                type="button"
                                onClick={() => setIsModalOpen(false)}
                                className="btn-header-refresh"
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                disabled={isSaving}
                                className="btn-header-new-route"
                            >
                                {isSaving ? 'Saving...' : editingQuery ? 'Update Query' : 'Save Query'}
                            </button>
                        </div>
                    </div>
                </form>
            </Modal>
        </div>
    );
}
