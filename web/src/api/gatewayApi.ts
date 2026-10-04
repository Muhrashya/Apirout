import { apiClient } from './apiClient';

export interface GatewayService {
    id: string;
    name: string;
    description?: string;
    protocol: 'http' | 'https' | 'grpc' | 'tcp' | 'database';
    host: string;
    port: number;
    path: string;
    connect_timeout: number;
    read_timeout: number;
    write_timeout: number;
    retries: number;
    enabled: boolean;
    upstream_id?: string;
    upstream_name?: string;
    connection_id?: string;
    connection_name?: string;
    routes_count?: number;
    created_at: string;
    updated_at: string;
}

export interface Upstream {
    id: string;
    name: string;
    algorithm: 'round-robin' | 'least-connections' | 'consistent-hashing' | 'latency';
    hash_on: string;
    hash_fallback: string;
    hash_on_cookie?: string;
    slots: number;
    enabled: boolean;
    health_check_enabled: boolean;
    health_check_path: string;
    health_check_interval: number;
    health_check_timeout: number;
    healthy_threshold: number;
    unhealthy_threshold: number;
    targets_count?: number;
    healthy_count?: number;
    targets?: Target[];
    created_at: string;
    updated_at: string;
}

export interface Target {
    id: string;
    upstream_id: string;
    target: string;
    host: string;
    port: number;
    weight: number;
    enabled: boolean;
    health_status: 'healthy' | 'unhealthy' | 'checking' | 'disabled' | 'unknown';
    failover: boolean;
    created_at: string;
    updated_at: string;
}

export interface Route {
    id: string;
    service_id?: string;
    service_name?: string;
    name: string;
    group_id?: string;
    group_name?: string;
    method: string;
    path: string;
    protocols: string;
    strip_path: boolean;
    preserve_host: boolean;
    regex_priority: number;
    https_redirect_status_code: number;
    query_id?: string;
    enabled: boolean;
    is_public: boolean;
    rate_limit_enabled: boolean;
    rate_limit_rpm: number;
    rate_limit_burst: number;
    default_limit: number;
    max_limit: number;
    return_total_count: boolean;
    allow_dynamic_filters: boolean;
    allowed_domains: string;
    allowed_ips: string;
    cors_origins: string;
    description?: string;
    created_by_email?: string;
    created_at: string;
    updated_at: string;
}

export interface Consumer {
    id: string;
    group_id?: string;
    group_name?: string;
    username: string;
    custom_id?: string;
    description?: string;
    status: 'active' | 'suspended' | 'revoked';
    allowed_endpoints_count?: number;
    credentials_count?: number;
    created_at: string;
    updated_at: string;
}

export interface ConsumerCredential {
    id: string;
    consumer_id: string;
    name: string;
    key_type: string;
    key_id: string;
    masked_key: string;
    enabled: boolean;
    expires_at?: string;
    last_used_at?: string;
    created_at: string;
}

export interface ConsumerCredentialCreated extends ConsumerCredential {
    key_secret: string;
}

export interface ConsumerGroup {
    id: string;
    name: string;
    description?: string;
    rate_limit_rpm: number;
    rate_limit_burst: number;
    consumers_count?: number;
    created_at: string;
    updated_at: string;
}

export interface GatewayDomain {
    id: string;
    domain_name: string;
    display_name: string;
    is_default: boolean;
    environment: string;
    default_allowed_ips?: string;
    description?: string | null;
    created_at: string;
    updated_at?: string;
}

export interface Connection {
    id: string;
    group_id?: string;
    name: string;
    db_type: string;
    host: string;
    port: number;
    db_name: string;
    username: string;
    ssl_mode?: string;
    connection_health?: string;
    max_open_conns?: number;
    max_idle_conns?: number;
    conn_max_lifetime?: number;
    is_read_only?: boolean;
    created_at: string;
}

export interface QueryItem {
    id: string;
    group_id?: string;
    connection_id: string;
    name: string;
    description?: string;
    sql_text: string;
    params_schema?: string;
    created_at: string;
}

// API methods
export const gatewayApi = {
    // Services
    getServices: () => apiClient.get<{ data: GatewayService[] }>('/services'),
    getService: (id: string) => apiClient.get<GatewayService>(`/services/${id}`),
    createService: (data: Partial<GatewayService>) => apiClient.post<GatewayService>('/services', data),
    updateService: (id: string, data: Partial<GatewayService>) => apiClient.put<GatewayService>(`/services/${id}`, data),
    deleteService: (id: string) => apiClient.delete(`/services/${id}`),
    getServiceRoutes: (id: string) => apiClient.get<{ data: Route[] }>(`/services/${id}/routes`),

    // Upstreams
    getUpstreams: () => apiClient.get<{ data: Upstream[] }>('/upstreams'),
    getUpstream: (id: string) => apiClient.get<Upstream>(`/upstreams/${id}`),
    createUpstream: (data: Partial<Upstream>) => apiClient.post<Upstream>('/upstreams', data),
    updateUpstream: (id: string, data: Partial<Upstream>) => apiClient.put<Upstream>(`/upstreams/${id}`, data),
    deleteUpstream: (id: string) => apiClient.delete(`/upstreams/${id}`),
    healthCheckUpstream: (id: string) => apiClient.post<{ results: any[] }>(`/upstreams/${id}/health-check`),

    // Targets
    getTargets: (upstreamId: string) => apiClient.get<{ data: Target[] }>(`/upstreams/${upstreamId}/targets`),
    createTarget: (upstreamId: string, data: Partial<Target>) => apiClient.post<Target>(`/upstreams/${upstreamId}/targets`, data),
    updateTarget: (targetId: string, data: Partial<Target>) => apiClient.put<Target>(`/targets/${targetId}`, data),
    deleteTarget: (targetId: string) => apiClient.delete(`/targets/${targetId}`),
    toggleTarget: (targetId: string, enabled: boolean) => apiClient.post(`/targets/${targetId}/toggle`, { enabled }),
    healthCheckTarget: (targetId: string) => apiClient.post<{ health_status: string; latency_ms: number; error: string }>(`/targets/${targetId}/health-check`),

    // Routes
    getRoutes: () => apiClient.get<{ data: Route[] }>('/routes'),
    getRoute: (id: string) => apiClient.get<Route>(`/routes/${id}`),
    createRoute: (data: Partial<Route>) => apiClient.post<Route>('/routes', data),
    updateRoute: (id: string, data: Partial<Route>) => apiClient.put(`/routes/${id}`, data),
    deleteRoute: (id: string) => apiClient.delete(`/routes/${id}`),

    // Gateway Domains (Host-based routing)
    getDomains: () => apiClient.get<{ data: GatewayDomain[] }>('/domains'),
    createDomain: (data: Partial<GatewayDomain>) => apiClient.post<GatewayDomain>('/domains', data),
    deleteDomain: (id: string) => apiClient.delete(`/domains/${id}`),

    // Consumers
    getConsumers: () => apiClient.get<{ data: Consumer[] }>('/consumers'),
    getConsumer: (id: string) => apiClient.get<Consumer>(`/consumers/${id}`),
    createConsumer: (data: Partial<Consumer> & { password?: string; endpoint_ids?: string[] }) => apiClient.post<Consumer>('/consumers', data),
    updateConsumer: (id: string, data: Partial<Consumer> & { password?: string; endpoint_ids?: string[] }) => apiClient.put<Consumer>(`/consumers/${id}`, data),
    deleteConsumer: (id: string) => apiClient.delete(`/consumers/${id}`),

    // Credentials
    getCredentials: (consumerId: string) => apiClient.get<{ data: ConsumerCredential[] }>(`/consumers/${consumerId}/credentials`),
    createCredential: (consumerId: string, data: { name: string; key_type: string; secret?: string; expires_at?: string }) =>
        apiClient.post<ConsumerCredentialCreated>(`/consumers/${consumerId}/credentials`, data),
    toggleCredential: (consumerId: string, credId: string, enabled: boolean) =>
        apiClient.post(`/consumers/${consumerId}/credentials/${credId}/toggle`, { enabled }),
    deleteCredential: (consumerId: string, credId: string) =>
        apiClient.delete(`/consumers/${consumerId}/credentials/${credId}`),

    // Consumer Routes
    getConsumerRoutes: (consumerId: string) => apiClient.get<{ route_ids: string[] }>(`/consumers/${consumerId}/routes`),
    updateConsumerRoutes: (consumerId: string, routeIds: string[]) => apiClient.put(`/consumers/${consumerId}/routes`, { route_ids: routeIds }),

    // Consumer Groups
    getConsumerGroups: () => apiClient.get<{ data: ConsumerGroup[] }>('/consumer-groups'),
    getConsumerGroup: (id: string) => apiClient.get<ConsumerGroup>(`/consumer-groups/${id}`),
    createConsumerGroup: (data: Partial<ConsumerGroup>) => apiClient.post<ConsumerGroup>('/consumer-groups', data),
    updateConsumerGroup: (id: string, data: Partial<ConsumerGroup>) => apiClient.put<ConsumerGroup>(`/consumer-groups/${id}`, data),
    deleteConsumerGroup: (id: string) => apiClient.delete(`/consumer-groups/${id}`),

    // Database Connections & Queries
    getConnections: () => apiClient.get<{ data: Connection[] }>('/connections'),
    getConnection: (id: string) => apiClient.get<Connection>(`/connections/${id}`),
    createConnection: (data: any) => apiClient.post<Connection>('/connections', data),
    updateConnection: (id: string, data: any) => apiClient.put(`/connections/${id}`, data),
    deleteConnection: (id: string) => apiClient.delete(`/connections/${id}`),
    testConnection: (id: string) => apiClient.post<{ success: boolean; latency_ms?: number; error?: string }>(`/connections/${id}/test`),
    testRawConnection: (data: any) => apiClient.post<{ success: boolean; latency_ms?: number; error?: string }>('/connections/test', data),
    getQueries: () => apiClient.get<{ data: QueryItem[] }>('/queries'),
    getQuery: (id: string) => apiClient.get<QueryItem>(`/queries/${id}`),
    createQuery: (data: any) => apiClient.post<QueryItem>('/queries', data),
    updateQuery: (id: string, data: any) => apiClient.put(`/queries/${id}`, data),
    deleteQuery: (id: string) => apiClient.delete(`/queries/${id}`),
    previewQuery: (data: { connection_id: string; sql: string; limit?: number; offset?: number }) =>
        apiClient.post<{ data: any[] }>('/queries/preview', data),

    // Auth Plugins (Inbound)
    getAuthPlugins: (params?: { scope?: string; entity_id?: string }) => {
        const q = new URLSearchParams();
        if (params?.scope) q.set('scope', params.scope);
        if (params?.entity_id) q.set('entity_id', params.entity_id);
        const qs = q.toString();
        return apiClient.get<{ data: AuthPluginConfig[] }>(`/auth-plugins${qs ? `?${qs}` : ''}`);
    },
    getAuthPlugin: (id: string) =>
        apiClient.get<AuthPluginConfig>(`/auth-plugins/${id}`),
    createAuthPlugin: (data: Partial<AuthPluginConfig>) =>
        apiClient.post<AuthPluginConfig>('/auth-plugins', data),
    updateAuthPlugin: (id: string, data: Partial<AuthPluginConfig>) =>
        apiClient.put<AuthPluginConfig>(`/auth-plugins/${id}`, data),
    deleteAuthPlugin: (id: string) =>
        apiClient.delete(`/auth-plugins/${id}`),

    // Outbound Auth (Upstream)
    getOutboundAuth: (serviceId: string) =>
        apiClient.get<ServiceOutboundAuth>(`/services/${serviceId}/outbound-auth`),
    setOutboundAuth: (serviceId: string, data: Partial<ServiceOutboundAuth>) =>
        apiClient.put<ServiceOutboundAuth>(`/services/${serviceId}/outbound-auth`, data),
    deleteOutboundAuth: (serviceId: string) =>
        apiClient.delete(`/services/${serviceId}/outbound-auth`),

    // Monitoring & Request Logs (Kong Analytics Explorer)
    getRequests: (params?: { page?: number; limit?: number; search?: string; method?: string; status?: string } | number) => {
        const q = new URLSearchParams();
        if (typeof params === 'number') {
            q.set('limit', String(params));
        } else if (params) {
            if (params.page) q.set('page', String(params.page));
            if (params.limit) q.set('limit', String(params.limit));
            if (params.search) q.set('search', params.search);
            if (params.method && params.method !== 'all') q.set('method', params.method);
            if (params.status && params.status !== 'all') q.set('status', params.status);
        }
        const queryStr = q.toString() ? `?${q.toString()}` : '';
        return apiClient.get<{ data: ApiRequest[]; pagination: PaginationMeta }>(`/monitoring/requests${queryStr}`);
    },
    getDashboardStats: () => apiClient.get<any>('/dashboard/stats'),
};

export interface PaginationMeta {
    total: number;
    page: number;
    limit: number;
    total_pages: number;
}

export interface AuthPluginConfig {
    id: string;
    plugin_name: string;
    scope: 'service' | 'route' | 'consumer' | 'global';
    service_id?: string;
    route_id?: string;
    consumer_id?: string;
    service_name?: string;
    route_name?: string;
    consumer_name?: string;
    config: string;
    enabled: boolean;
    created_at?: string;
    updated_at?: string;
}

export interface ServiceOutboundAuth {
    id?: string;
    service_id: string;
    auth_type: string;
    config: string | Record<string, any>;
    enabled: boolean;
    created_at?: string;
    updated_at?: string;
}

export interface ApiRequest {
    id: string;
    timestamp: string;
    event_type: string;
    user_id?: string | null;
    api_client_id?: string | null;
    username: string;
    endpoint_id?: string | null;
    ip_address?: string | null;
    details: string;
    success: boolean;
    duration_ms?: number | null;
    error_message?: string | null;
    method: string;
    path: string;
    status_code: number;
    client_ip: string;
    user_agent?: string;
    host?: string;
    size?: number;
    route_name?: string;
    service_name?: string;
    upstream_uri?: string;
    auth_type?: string;
    gateway_latency_ms?: number;
    upstream_latency_ms?: number;
}
