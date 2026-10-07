import { useState, useEffect } from 'react';
import { apiClient } from '../api/apiClient';
import { useToast } from '../components/Toast';

interface Endpoint {
    id: string;
    path: string;
    method: string;
}

export default function Docs() {
    const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
    const [isLoading, setIsLoading] = useState(true);
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
                    id: details.operationId,
                    path,
                    method: method.toUpperCase()
                }))
            );
            setEndpoints(eps);
        } catch (err) {
            toast.error('Failed to load API documentation');
        } finally {
            setIsLoading(false);
        }
    };

    const toggleSelectAll = (checked: boolean) => {
        if (checked) {
            setSelectedIds(new Set(endpoints.map(ep => ep.id)));
        } else {
            setSelectedIds(new Set());
        }
    };

    const toggleSelect = (id: string) => {
        const newSelected = new Set(selectedIds);
        if (newSelected.has(id)) {
            newSelected.delete(id);
        } else {
            newSelected.add(id);
        }
        setSelectedIds(newSelected);
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
            a.download = 'apirout-collection.json';
            a.click();
            URL.revokeObjectURL(url);
            toast.success('Postman collection downloaded');
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
            a.download = 'openapi.json';
            a.click();
            URL.revokeObjectURL(url);
            toast.success('OpenAPI spec downloaded');
        } catch (err) {
            toast.error('Failed to download OpenAPI spec');
        }
    };

    if (isLoading) {
        return <div className="loading"><div className="spinner"></div></div>;
    }

    return (
        <div>
            <div className="page-header">
                <h1 className="page-title">API Documentation</h1>
                <div className="flex gap-2">
                    <button className="btn btn-secondary" onClick={downloadOpenAPI}>
                        Download OpenAPI
                        {selectedIds.size > 0 && selectedIds.size < endpoints.length && ` (${selectedIds.size})`}
                    </button>
                    <button className="btn btn-primary" onClick={downloadPostman}>
                        Download Postman
                        {selectedIds.size > 0 && selectedIds.size < endpoints.length && ` (${selectedIds.size})`}
                    </button>
                </div>
            </div>

            <div className="card">
                <div className="card-header flex justify-between items-center">
                    <h2 className="card-title">Published Endpoints</h2>
                    <span className="text-sm text-muted">
                        {selectedIds.size} of {endpoints.length} selected
                    </span>
                </div>

                {endpoints.length > 0 ? (
                    <table className="table">
                        <thead>
                            <tr>
                                <th style={{ width: '40px' }}>
                                    <input
                                        type="checkbox"
                                        className="checkbox"
                                        checked={endpoints.length > 0 && selectedIds.size === endpoints.length}
                                        onChange={(e) => toggleSelectAll(e.target.checked)}
                                    />
                                </th>
                                <th>Method</th>
                                <th>Path</th>
                            </tr>
                        </thead>
                        <tbody>
                            {endpoints.map((ep) => (
                                <tr key={ep.id} className={selectedIds.has(ep.id) ? 'bg-base-200' : ''}>
                                    <td>
                                        <input
                                            type="checkbox"
                                            className="checkbox"
                                            checked={selectedIds.has(ep.id)}
                                            onChange={() => toggleSelect(ep.id)}
                                        />
                                    </td>
                                    <td>
                                        <span className="badge badge-info">{ep.method}</span>
                                    </td>
                                    <td><code>{ep.path}</code></td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                ) : (
                    <div className="empty-state">
                        <h3>No endpoints published</h3>
                        <p>Publish endpoints to see them in the documentation</p>
                    </div>
                )}
            </div>

            <div className="card mt-4">
                <div className="card-header">
                    <h2 className="card-title">Authentication</h2>
                </div>
                <div style={{ padding: '0 24px 24px' }}>
                    <h4 style={{ marginBottom: '12px' }}>Admin Authentication (Email only, not Username)</h4>
                    <pre style={{ background: 'var(--color-bg)', padding: '16px', borderRadius: '8px', overflow: 'auto' }}>
                        {`POST /auth/login
Content-Type: application/json

{
  "email": "admin@example.com",
  "password": "Admin123!"
}

Response:
{
  "token": "eyJ...",
  "expires_at": "2024-12-25T00:00:00Z",
  "user": { "id": "...", "email": "...", "role": "SUPER_ADMIN" }
}`}
                    </pre>

                    <h4 style={{ margin: '24px 0 12px' }}>API Client Authentication</h4>
                    <pre style={{ background: 'var(--color-bg)', padding: '16px', borderRadius: '8px', overflow: 'auto' }}>
                        {`POST /api/login
Content-Type: application/json

{
  "username": "my_client", // or "client@example.com"
  "password": "secret"
}

Response:
{
  "status": true,
  "message": "Success",
  "data": {
    "token": "eyJ...",
    "expired": "2026-01-20 16:58:39"
  }
}`}
                    </pre>

                    <h4 style={{ margin: '24px 0 12px' }}>Using the Token (3 Methods Supported)</h4>
                    <pre style={{ background: 'var(--color-bg)', padding: '16px', borderRadius: '8px', overflow: 'auto' }}>
                        {`# Method 1: Authorization Bearer Header (recommended)
GET /api/your-endpoint
Authorization: Bearer <token>

# Method 2: X-API-Token Header
GET /api/your-endpoint
X-API-Token: <token>

# Method 3: Query Parameter (less secure, but convenient)
GET /api/your-endpoint?token=<token>`}
                    </pre>
                </div>
            </div>
        </div>
    );
}

