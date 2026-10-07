const API_BASE = '/api/v1';

// Read token fresh on each request (fixes SPA auth bug)
function getToken(): string | null {
    return localStorage.getItem('token');
}

async function request<T>(
    method: string,
    path: string,
    data?: unknown
): Promise<T> {
    const token = getToken(); // Read fresh each time

    const headers: HeadersInit = {
        'Content-Type': 'application/json',
    };

    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    let response: Response;
    try {
        response = await fetch(`${API_BASE}${path}`, {
            method,
            headers,
            body: data ? JSON.stringify(data) : undefined,
        });
    } catch (err) {
        // Network error (server down, CORS, etc.)
        throw new Error(`Network error: ${err instanceof Error ? err.message : 'Failed to connect'}`);
    }

    let json: { error?: string; message?: string } & T;
    try {
        json = await response.json();
    } catch {
        // Response not JSON
        if (!response.ok) {
            if (response.status === 401) {
                clearToken();
                if (window.location.pathname !== '/login') {
                    window.location.href = '/login';
                }
            }
            throw new Error(`Server error: ${response.status} ${response.statusText}`);
        }
        throw new Error('Invalid JSON response from server');
    }

    if (!response.ok) {
        if (response.status === 401) {
            clearToken();
            if (window.location.pathname !== '/login') {
                window.location.href = '/login';
            }
        }
        throw new Error(json.message || json.error || `Request failed: ${response.status}`);
    }

    return json;
}

export const apiClient = {
    get: <T>(path: string) => request<T>('GET', path),
    post: <T>(path: string, data?: unknown) => request<T>('POST', path, data),
    put: <T>(path: string, data?: unknown) => request<T>('PUT', path, data),
    patch: <T>(path: string, data?: unknown) => request<T>('PATCH', path, data),
    delete: <T>(path: string) => request<T>('DELETE', path),
};

export function setToken(token: string) {
    localStorage.setItem('token', token);
}

export function clearToken() {
    localStorage.removeItem('token');
}

export function hasToken(): boolean {
    return !!localStorage.getItem('token');
}
