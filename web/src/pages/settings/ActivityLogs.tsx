import { useState, useEffect } from 'react';
import { apiClient } from '../../api/apiClient';
import { useToast } from '../../components/Toast';

interface AuditLog {
    id: string;
    event_type: string;
    user_id: string | null;
    api_client_id: string | null;
    endpoint_id: string | null;
    ip_address: string | null;
    details: string;
    success: boolean;
    duration_ms: number | null;
    error_message: string | null;
    timestamp: string;
}

export default function ActivityLogs() {
    const [logs, setLogs] = useState<AuditLog[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [filter, setFilter] = useState('');
    const [searchTerm, setSearchTerm] = useState('');
    const toast = useToast();

    useEffect(() => {
        loadLogs();
    }, [filter]);

    const loadLogs = async () => {
        setIsLoading(true);
        try {
            const queryParam = filter ? `?event_type=${filter}` : '';
            const response = await apiClient.get<{ data: AuditLog[] }>(`/audit-logs${queryParam}`);
            setLogs(response.data ?? []);
        } catch (err) {
            toast.error('Failed to load activity logs');
        } finally {
            setIsLoading(false);
        }
    };

    const eventTypes = [
        { value: '', label: 'All Event Types' },
        { value: 'admin_login_success', label: 'Admin Login Success' },
        { value: 'admin_login_failed', label: 'Admin Login Failed' },
        { value: 'api_client_login_success', label: 'API Client Login Success' },
        { value: 'api_client_login_failed', label: 'API Client Login Failed' },
        { value: 'route_create', label: 'Route Created' },
        { value: 'route_update', label: 'Route Updated' },
        { value: 'route_delete', label: 'Route Deleted' },
        { value: 'service_create', label: 'Service Created' },
        { value: 'service_update', label: 'Service Updated' },
        { value: 'service_delete', label: 'Service Deleted' },
        { value: 'upstream_create', label: 'Upstream Created' },
        { value: 'upstream_update', label: 'Upstream Updated' },
        { value: 'upstream_delete', label: 'Upstream Deleted' },
        { value: 'consumer_create', label: 'Consumer Created' },
        { value: 'consumer_delete', label: 'Consumer Deleted' },
        { value: 'endpoint_exec', label: 'Endpoint Execution' },
    ];

    const filteredLogs = logs.filter(log => {
        if (!searchTerm) return true;
        const s = searchTerm.toLowerCase();
        return (
            log.event_type.toLowerCase().includes(s) ||
            (log.ip_address && log.ip_address.toLowerCase().includes(s)) ||
            (log.details && log.details.toLowerCase().includes(s)) ||
            (log.error_message && log.error_message.toLowerCase().includes(s))
        );
    });

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Console</span>
                <span className="breadcrumb-sep">/</span>
                <span>Audit Trail</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Activity Logs</span>
            </div>

            {/* Header */}
            <div className="overview-header" style={{ marginBottom: '20px' }}>
                <div className="overview-title-col">
                    <h1 className="overview-title">System Activity Logs</h1>
                    <p className="overview-subtitle">
                        Immutable administrative audit trail recording console authentication, config changes, and security events.
                    </p>
                </div>
                <button
                    className="btn btn-secondary"
                    onClick={loadLogs}
                    style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
                >
                    Refresh Logs
                </button>
            </div>

            {/* Filter Bar */}
            <div style={{
                backgroundColor: '#ffffff',
                border: '1px solid #e2e8f0',
                borderRadius: '10px',
                padding: '12px 16px',
                marginBottom: '16px',
                display: 'flex',
                gap: '12px',
                alignItems: 'center',
                flexWrap: 'wrap'
            }}>
                <div style={{ flex: 1, minWidth: '220px' }}>
                    <input
                        type="text"
                        className="input"
                        placeholder="Filter by keyword, IP, or details..."
                        value={searchTerm}
                        onChange={e => setSearchTerm(e.target.value)}
                        style={{ width: '100%' }}
                    />
                </div>
                <div style={{ minWidth: '200px' }}>
                    <select
                        className="select select-filter"
                        value={filter}
                        onChange={(e) => setFilter(e.target.value)}
                        style={{ width: '100%' }}
                    >
                        {eventTypes.map((type) => (
                            <option key={type.value} value={type.value}>
                                {type.label}
                            </option>
                        ))}
                    </select>
                </div>
                <div style={{ fontSize: '13px', color: '#64748b', marginLeft: 'auto' }}>
                    Showing {filteredLogs.length} of {logs.length} records
                </div>
            </div>

            {/* Logs Table Card */}
            <div className="card" style={{ padding: 0, overflow: 'hidden', border: '1px solid #e2e8f0', borderRadius: '12px' }}>
                {isLoading ? (
                    <div className="loading" style={{ padding: '60px 0' }}>
                        <div className="spinner"></div>
                    </div>
                ) : filteredLogs.length > 0 ? (
                    <div style={{ overflowX: 'auto' }}>
                        <table className="table" style={{ width: '100%', borderCollapse: 'collapse' }}>
                            <thead>
                                <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                    <th style={{ padding: '12px 16px', textAlign: 'left', fontSize: '12px', fontWeight: 600, color: '#475569' }}>Timestamp</th>
                                    <th style={{ padding: '12px 16px', textAlign: 'left', fontSize: '12px', fontWeight: 600, color: '#475569' }}>Event Type</th>
                                    <th style={{ padding: '12px 16px', textAlign: 'left', fontSize: '12px', fontWeight: 600, color: '#475569' }}>Origin IP</th>
                                    <th style={{ padding: '12px 16px', textAlign: 'left', fontSize: '12px', fontWeight: 600, color: '#475569' }}>Status</th>
                                    <th style={{ padding: '12px 16px', textAlign: 'left', fontSize: '12px', fontWeight: 600, color: '#475569' }}>Duration</th>
                                    <th style={{ padding: '12px 16px', textAlign: 'left', fontSize: '12px', fontWeight: 600, color: '#475569' }}>Details</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredLogs.map((log) => (
                                    <tr key={log.id} style={{ borderBottom: '1px solid #f1f5f9', fontSize: '13px' }}>
                                        <td style={{ padding: '12px 16px', whiteSpace: 'nowrap', color: '#64748b' }}>
                                            {new Date(log.timestamp).toLocaleString()}
                                        </td>
                                        <td style={{ padding: '12px 16px' }}>
                                            <code style={{
                                                fontSize: '12px',
                                                padding: '2px 6px',
                                                backgroundColor: '#f1f5f9',
                                                borderRadius: '4px',
                                                color: '#0f172a'
                                            }}>
                                                {log.event_type}
                                            </code>
                                        </td>
                                        <td style={{ padding: '12px 16px' }}>
                                            <span style={{ fontFamily: 'monospace', fontSize: '12.5px', color: '#334155' }}>
                                                {log.ip_address || '-'}
                                            </span>
                                        </td>
                                        <td style={{ padding: '12px 16px' }}>
                                            <span className={`badge ${log.success ? 'badge-success' : 'badge-danger'}`} style={{
                                                fontSize: '11px',
                                                padding: '2px 8px',
                                                borderRadius: '12px'
                                            }}>
                                                {log.success ? 'Success' : 'Failed'}
                                            </span>
                                        </td>
                                        <td style={{ padding: '12px 16px', color: '#64748b' }}>
                                            {log.duration_ms !== null ? `${log.duration_ms}ms` : '-'}
                                        </td>
                                        <td style={{ padding: '12px 16px', maxWidth: '350px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: '#475569' }} title={log.error_message || log.details}>
                                            {log.error_message || log.details || '-'}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                ) : (
                    <div className="empty-state" style={{ padding: '48px 16px', textAlign: 'center' }}>
                        <div style={{ fontSize: '32px', marginBottom: '8px' }}>📋</div>
                        <h3 style={{ fontSize: '16px', fontWeight: 600, color: '#0f172a', margin: '0 0 4px 0' }}>No activity records found</h3>
                        <p style={{ fontSize: '13px', color: '#64748b', margin: 0 }}>
                            {searchTerm ? 'No activity matches your filter.' : 'Administrative audit events will appear here as they occur.'}
                        </p>
                    </div>
                )}
            </div>
        </div>
    );
}
