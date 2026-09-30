import { useState, useEffect } from 'react';
import { apiClient } from '../api/apiClient';
import { useToast } from '../components/Toast';

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

export default function AuditLogs() {
    const [logs, setLogs] = useState<AuditLog[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [filter, setFilter] = useState('');
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
            toast.error('Failed to load audit logs');
        } finally {
            setIsLoading(false);
        }
    };

    const eventTypes = [
        { value: '', label: 'All Events' },
        { value: 'admin_login_success', label: 'Admin Login Success' },
        { value: 'admin_login_failed', label: 'Admin Login Failed' },
        { value: 'api_client_login_success', label: 'API Client Login Success' },
        { value: 'api_client_login_failed', label: 'API Client Login Failed' },
        { value: 'endpoint_exec', label: 'Endpoint Execution' },
    ];

    return (
        <div>
            <div className="page-header">
                <h1 className="page-title">Audit Logs</h1>
                <div className="flex gap-2">
                    <select
                        className="select select-filter"
                        value={filter}
                        onChange={(e) => setFilter(e.target.value)}
                    >
                        {eventTypes.map((type) => (
                            <option key={type.value} value={type.value}>
                                {type.label}
                            </option>
                        ))}
                    </select>
                    <button className="btn btn-secondary" onClick={loadLogs}>
                        Refresh
                    </button>
                </div>
            </div>

            <div className="card">
                {isLoading ? (
                    <div className="loading"><div className="spinner"></div></div>
                ) : logs.length > 0 ? (
                    <table className="table">
                        <thead>
                            <tr>
                                <th>Time</th>
                                <th>Event</th>
                                <th>IP Address</th>
                                <th>Status</th>
                                <th>Duration</th>
                                <th>Details</th>
                            </tr>
                        </thead>
                        <tbody>
                            {logs.map((log) => (
                                <tr key={log.id}>
                                    <td style={{ whiteSpace: 'nowrap' }}>
                                        {new Date(log.timestamp).toLocaleString()}
                                    </td>
                                    <td>
                                        <code>{log.event_type}</code>
                                    </td>
                                    <td>
                                        <code>{log.ip_address || '-'}</code>
                                    </td>
                                    <td>
                                        <span className={`badge ${log.success ? 'badge-success' : 'badge-danger'}`}>
                                            {log.success ? 'Success' : 'Failed'}
                                        </span>
                                    </td>
                                    <td>
                                        {log.duration_ms !== null ? `${log.duration_ms}ms` : '-'}
                                    </td>
                                    <td style={{ maxWidth: '300px', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                        {log.error_message || log.details || '-'}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                ) : (
                    <div className="empty-state">
                        <h3>No audit logs</h3>
                        <p>Events will appear here as they occur</p>
                    </div>
                )}
            </div>
        </div>
    );
}
