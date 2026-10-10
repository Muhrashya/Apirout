import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { gatewayApi, Connection } from '../../api/gatewayApi';
import { useToast } from '../../components/Toast';
import { DatabaseEnginePickerModal } from '../../components/DatabaseEnginePickerModal';
import {
    IconDatabase,
    IconSearch,
    IconRefresh,
    IconPlus,
    IconPencil,
    IconTrash,
    IconBolt,
    IconCheck,
    IconAlertTriangle,
} from '../../components/Icons';

export default function ConnectionsPage() {
    const navigate = useNavigate();
    const toast = useToast();

    const [connections, setConnections] = useState<Connection[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const [typeFilter, setTypeFilter] = useState('all');
    const [isPickerModalOpen, setIsPickerModalOpen] = useState(false);

    // Testing state for rows
    const [testingId, setTestingId] = useState<string | null>(null);
    const [testResults, setTestResults] = useState<Record<string, { success: boolean; latency_ms?: number; error?: string }>>({});

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setIsLoading(true);
        try {
            const res = await gatewayApi.getConnections();
            const list = Array.isArray(res.data) ? res.data : [];
            setConnections(list);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load database connections');
        } finally {
            setIsLoading(false);
        }
    };

    const handleDeleteConnection = async (conn: Connection) => {
        if (!window.confirm(`Are you sure you want to delete connection "${conn.name}"?`)) return;
        try {
            await gatewayApi.deleteConnection(conn.id);
            toast.success(`Connection "${conn.name}" deleted`);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to delete connection');
        }
    };

    const handleTestSavedConnection = async (conn: Connection) => {
        setTestingId(conn.id);
        try {
            const res = await gatewayApi.testConnection(conn.id);
            setTestResults(prev => ({ ...prev, [conn.id]: res }));
            if (res.success) {
                toast.success(`${conn.name} is reachable (${res.latency_ms ?? 0}ms)`);
            } else {
                toast.error(res.error || `${conn.name} unreachable`);
            }
        } catch (err: any) {
            setTestResults(prev => ({ ...prev, [conn.id]: { success: false, error: err.message || 'Test failed' } }));
            toast.error(err.message || 'Test failed');
        } finally {
            setTestingId(null);
        }
    };

    // Filter connections
    const filteredConnections = connections.filter(conn => {
        const matchesSearch =
            conn.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            conn.host.toLowerCase().includes(searchTerm.toLowerCase()) ||
            conn.db_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            conn.username.toLowerCase().includes(searchTerm.toLowerCase()) ||
            conn.db_type.toLowerCase().includes(searchTerm.toLowerCase());

        const matchesType =
            typeFilter === 'all' ||
            (typeFilter === 'postgres' && (conn.db_type === 'postgres' || conn.db_type === 'postgresql')) ||
            conn.db_type === typeFilter;

        return matchesSearch && matchesType;
    });

    const getDbTypeBadge = (dbType: string) => {
        const t = dbType.toLowerCase();
        if (t === 'postgres' || t === 'postgresql') {
            return { label: 'PostgreSQL', bg: '#eff6ff', color: '#2563eb', border: '#bfdbfe' };
        }
        if (t === 'mysql') {
            return { label: 'MySQL', bg: '#fff7ed', color: '#ea580c', border: '#fed7aa' };
        }
        if (t === 'mssql' || t === 'sqlserver') {
            return { label: 'SQL Server', bg: '#faf5ff', color: '#9333ea', border: '#e9d5ff' };
        }
        if (t === 'mongodb' || t === 'mongo') {
            return { label: 'MongoDB', bg: '#f0fdf4', color: '#16a34a', border: '#bbf7d0' };
        }
        if (t === 'redis') {
            return { label: 'Redis', bg: '#fef2f2', color: '#dc2626', border: '#fecaca' };
        }
        return { label: dbType, bg: '#f1f5f9', color: '#475569', border: '#cbd5e1' };
    };

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Settings</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Database Sources</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Database Sources</h1>
                    <p className="overview-subtitle">
                        Configure database connection profiles (PostgreSQL, MySQL, SQL Server) for Instant API generation, SQL query runners, and dynamic data proxying.
                    </p>
                </div>

                <div className="overview-header-actions">
                    <button className="btn-header-refresh" onClick={loadData} title="Reload connections">
                        <IconRefresh width={14} height={14} />
                        <span>Refresh</span>
                    </button>
                    <button className="btn-header-new-route" onClick={() => setIsPickerModalOpen(true)}>
                        <IconPlus width={15} height={15} />
                        <span>New Connection</span>
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
                    {/* Search Input */}
                    <div style={{ position: 'relative', flex: '1 1 240px', minWidth: '200px' }}>
                        <IconSearch
                            width={15}
                            height={15}
                            style={{
                                position: 'absolute',
                                left: '12px',
                                top: '50%',
                                transform: 'translateY(-50%)',
                                color: '#94a3b8'
                            }}
                        />
                        <input
                            type="text"
                            placeholder="Filter by name, host, database name..."
                            value={searchTerm}
                            onChange={e => setSearchTerm(e.target.value)}
                            style={{
                                width: '100%',
                                padding: '8px 12px 8px 36px',
                                fontSize: '13.5px',
                                border: '1px solid #cbd5e1',
                                borderRadius: '8px',
                                outline: 'none',
                                boxSizing: 'border-box'
                            }}
                        />
                    </div>

                    {/* DB Type Filter */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <span style={{ fontSize: '13px', color: '#64748b', fontWeight: 500 }}>Type:</span>
                        <select
                            className="select select-filter"
                            value={typeFilter}
                            onChange={e => setTypeFilter(e.target.value)}
                        >
                            <option value="all">All Types</option>
                            <option value="postgres">PostgreSQL</option>
                            <option value="mysql">MySQL</option>
                            <option value="mongodb">MongoDB</option>
                            <option value="mssql">SQL Server</option>
                        </select>
                    </div>
                </div>

                <div style={{ fontSize: '13px', color: '#64748b' }}>
                    Showing <strong>{filteredConnections.length}</strong> of <strong>{connections.length}</strong> connections
                </div>
            </div>

            {/* Connections Table Card */}
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
                        <div style={{ color: '#64748b', fontSize: '13.5px' }}>Loading database sources...</div>
                    </div>
                ) : filteredConnections.length === 0 ? (
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
                            <IconDatabase width={24} height={24} />
                        </div>
                        <h3 style={{ fontSize: '16px', fontWeight: 600, color: '#0f172a', marginBottom: '6px' }}>
                            {searchTerm || typeFilter !== 'all' ? 'No Matching Connections' : 'No Database Sources Configured'}
                        </h3>
                        <p style={{ fontSize: '13.5px', color: '#64748b', maxWidth: '420px', margin: '0 auto 20px auto' }}>
                            {searchTerm || typeFilter !== 'all'
                                ? 'Try refining your search terms or filter selection.'
                                : 'Connect your PostgreSQL, MySQL, or SQL Server databases to enable automated Instant REST API endpoints and SQL execution.'}
                        </p>
                        {(!searchTerm && typeFilter === 'all') && (
                            <button
                                className="btn-header-new-route"
                                onClick={() => setIsPickerModalOpen(true)}
                            >
                                <IconPlus width={15} height={15} />
                                <span>Add Database Connection</span>
                            </button>
                        )}
                    </div>
                ) : (
                    <div style={{ overflowX: 'auto' }}>
                        <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '13.5px' }}>
                            <thead>
                                <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Connection Name</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Engine</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Host &amp; Port</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Database</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>User</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Status</th>
                                    <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569', textAlign: 'right' }}>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredConnections.map((conn, idx) => {
                                    const badge = getDbTypeBadge(conn.db_type);
                                    const testResult = testResults[conn.id];
                                    const isTesting = testingId === conn.id;

                                    return (
                                        <tr
                                            key={conn.id}
                                            style={{
                                                borderTop: idx > 0 ? '1px solid #f1f5f9' : 'none',
                                                backgroundColor: idx % 2 === 0 ? '#ffffff' : '#fafafa',
                                                transition: 'background-color 100ms ease'
                                            }}
                                            onMouseEnter={e => { e.currentTarget.style.backgroundColor = '#f8fafc'; }}
                                            onMouseLeave={e => { e.currentTarget.style.backgroundColor = idx % 2 === 0 ? '#ffffff' : '#fafafa'; }}
                                        >
                                            <td style={{ padding: '14px 20px' }}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                                    <div style={{
                                                        width: '32px',
                                                        height: '32px',
                                                        borderRadius: '8px',
                                                        backgroundColor: badge.bg,
                                                        color: badge.color,
                                                        display: 'flex',
                                                        alignItems: 'center',
                                                        justifyContent: 'center',
                                                        flexShrink: 0
                                                    }}>
                                                        <IconDatabase width={16} height={16} />
                                                    </div>
                                                    <div>
                                                        <div style={{ fontWeight: 600, color: '#0f172a' }}>{conn.name}</div>
                                                        <div style={{ fontSize: '12px', color: '#94a3b8' }}>ID: {conn.id.slice(0, 8)}...</div>
                                                    </div>
                                                </div>
                                            </td>

                                            <td style={{ padding: '14px 20px' }}>
                                                <span style={{
                                                    fontSize: '11.5px',
                                                    fontWeight: 600,
                                                    padding: '3px 8px',
                                                    borderRadius: '6px',
                                                    backgroundColor: badge.bg,
                                                    color: badge.color,
                                                    border: `1px solid ${badge.border}`
                                                }}>
                                                    {badge.label}
                                                </span>
                                            </td>

                                            <td style={{ padding: '14px 20px' }}>
                                                <code style={{
                                                    fontFamily: 'monospace',
                                                    fontSize: '12.5px',
                                                    backgroundColor: '#f1f5f9',
                                                    padding: '2px 6px',
                                                    borderRadius: '4px',
                                                    color: '#0f172a'
                                                }}>
                                                    {conn.host}:{conn.port}
                                                </code>
                                            </td>

                                            <td style={{ padding: '14px 20px', color: '#334155', fontWeight: 500 }}>
                                                {conn.db_name}
                                            </td>

                                            <td style={{ padding: '14px 20px', color: '#64748b' }}>
                                                {conn.username}
                                            </td>

                                            <td style={{ padding: '14px 20px' }}>
                                                {testResult ? (
                                                    testResult.success ? (
                                                        <span style={{
                                                            fontSize: '11px',
                                                            fontWeight: 600,
                                                            padding: '2px 8px',
                                                            borderRadius: '12px',
                                                            backgroundColor: '#dcfce7',
                                                            color: '#16a34a',
                                                            display: 'inline-flex',
                                                            alignItems: 'center',
                                                            gap: '4px'
                                                        }}>
                                                            <IconCheck width={12} height={12} />
                                                            <span>Online ({testResult.latency_ms ?? 0}ms)</span>
                                                        </span>
                                                    ) : (
                                                        <span style={{
                                                            fontSize: '11px',
                                                            fontWeight: 600,
                                                            padding: '2px 8px',
                                                            borderRadius: '12px',
                                                            backgroundColor: '#fee2e2',
                                                            color: '#dc2626',
                                                            display: 'inline-flex',
                                                            alignItems: 'center',
                                                            gap: '4px'
                                                        }} title={testResult.error}>
                                                            <IconAlertTriangle width={12} height={12} />
                                                            <span>Failed</span>
                                                        </span>
                                                    )
                                                ) : (
                                                    <span style={{
                                                        fontSize: '11px',
                                                        fontWeight: 600,
                                                        padding: '2px 8px',
                                                        borderRadius: '12px',
                                                        backgroundColor: conn.connection_health === 'healthy' ? '#dcfce7' : '#f1f5f9',
                                                        color: conn.connection_health === 'healthy' ? '#16a34a' : '#475569'
                                                    }}>
                                                        {conn.connection_health || 'Ready'}
                                                    </span>
                                                )}
                                            </td>

                                            <td style={{ padding: '14px 20px', textAlign: 'right' }}>
                                                <div style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
                                                    {/* Test Connection Button */}
                                                    <button
                                                        onClick={() => handleTestSavedConnection(conn)}
                                                        disabled={isTesting}
                                                        title="Test Connection Ping"
                                                        style={{
                                                            border: '1px solid #cbd5e1',
                                                            backgroundColor: '#ffffff',
                                                            padding: '5px 9px',
                                                            borderRadius: '6px',
                                                            color: '#334155',
                                                            fontSize: '12px',
                                                            fontWeight: 500,
                                                            cursor: isTesting ? 'wait' : 'pointer',
                                                            display: 'inline-flex',
                                                            alignItems: 'center',
                                                            gap: '4px'
                                                        }}
                                                    >
                                                        {isTesting ? (
                                                            <span style={{ fontSize: '11px', color: '#64748b' }}>Testing...</span>
                                                        ) : (
                                                            <>
                                                                <IconRefresh width={12} height={12} />
                                                                <span>Test</span>
                                                            </>
                                                        )}
                                                    </button>

                                                    {/* Go to Instant API Button */}
                                                    <button
                                                        onClick={() => navigate('/instant-api')}
                                                        title="Generate Instant REST API from this database"
                                                        style={{
                                                            border: '1px solid #bfdbfe',
                                                            backgroundColor: '#eff6ff',
                                                            padding: '5px 9px',
                                                            borderRadius: '6px',
                                                            color: '#2563eb',
                                                            fontSize: '12px',
                                                            fontWeight: 600,
                                                            cursor: 'pointer',
                                                            display: 'inline-flex',
                                                            alignItems: 'center',
                                                            gap: '4px'
                                                        }}
                                                    >
                                                        <IconBolt width={12} height={12} />
                                                        <span>Instant API</span>
                                                    </button>

                                                    {/* Edit Button */}
                                                    <button
                                                        onClick={() => navigate(`/settings/connections/${conn.id}/edit`)}
                                                        title="Edit Connection Profile"
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

                                                    {/* Delete Button */}
                                                    <button
                                                        onClick={() => handleDeleteConnection(conn)}
                                                        title="Delete Connection"
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

            {/* DBeaver-Style Database Engine Selector Modal */}
            <DatabaseEnginePickerModal
                isOpen={isPickerModalOpen}
                onClose={() => setIsPickerModalOpen(false)}
                onSelect={(engineId) => {
                    setIsPickerModalOpen(false);
                    navigate(`/settings/connections/new?engine=${engineId}`);
                }}
            />
        </div>
    );
}
