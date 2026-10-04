import { useState, useEffect } from 'react';
import { gatewayApi } from '../../api/gatewayApi';
import { useToast } from '../../components/Toast';
import { IconRefresh, IconRoutes, IconConsumers } from '../../components/Icons';

export default function AnalyticsPage() {
    const [stats, setStats] = useState<any>(null);
    const [isLoading, setIsLoading] = useState(true);
    const toast = useToast();

    useEffect(() => {
        loadAnalytics();
    }, []);

    const loadAnalytics = async () => {
        setIsLoading(true);
        try {
            const data = await gatewayApi.getDashboardStats();
            setStats(data || {});
        } catch (err: any) {
            toast.error(err.message || 'Failed to load analytics');
        } finally {
            setIsLoading(false);
        }
    };

    const avgDuration = stats?.avg_duration_ms ?? 0;
    const topEndpoints = stats?.top_endpoints || [];
    const topUsers = stats?.top_users || [];

    // Real statistical percentiles calculated directly from recorded request durations
    const p50 = stats?.latency_p50_ms ?? 0;
    const p90 = stats?.latency_p90_ms ?? 0;
    const p99 = stats?.latency_p99_ms ?? 0;

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            {/* Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', color: '#64748b', marginBottom: '4px' }}>
                        <span>Observability</span>
                        <span>/</span>
                        <span style={{ color: '#0f172a', fontWeight: 600 }}>Analytics</span>
                    </div>
                    <h1 style={{ fontSize: '24px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                        Gateway Analytics &amp; Latency Profiles
                    </h1>
                </div>

                <button className="btn-header-refresh" onClick={loadAnalytics} disabled={isLoading}>
                    <IconRefresh width={14} height={14} />
                    <span>Refresh</span>
                </button>
            </div>

            {/* Latency Percentiles Cards */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
                <PercentileCard title="Average Latency" value={`${avgDuration} ms`} label="Mean Response Time" />
                <PercentileCard title="P50 Latency" value={`${p50} ms`} label="50% of requests faster than" highlight="#2563eb" />
                <PercentileCard title="P90 Latency" value={`${p90} ms`} label="90% of requests faster than" highlight="#d97706" />
                <PercentileCard title="P99 Latency" value={`${p99} ms`} label="99% of requests faster than" highlight="#dc2626" />
            </div>

            {/* Top Endpoints & Top Consumers Tables */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
                {/* Top Routes by Volume */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        <IconRoutes width={18} height={18} style={{ color: '#2563eb' }} />
                        <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: 0 }}>Top Routes by Volume (24h)</h2>
                    </div>

                    {topEndpoints.length === 0 ? (
                        <div style={{ textAlign: 'center', padding: '40px 0', color: '#94a3b8', fontSize: '13px' }}>
                            No route requests recorded yet.
                        </div>
                    ) : (
                        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
                            <thead>
                                <tr style={{ borderBottom: '1px solid #e2e8f0', color: '#64748b' }}>
                                    <th style={{ padding: '8px 10px' }}>Route Path</th>
                                    <th style={{ padding: '8px 10px' }}>Requests</th>
                                    <th style={{ padding: '8px 10px' }}>Avg Latency</th>
                                </tr>
                            </thead>
                            <tbody>
                                {topEndpoints.map((ep: any, idx: number) => (
                                    <tr key={idx} style={{ borderBottom: '1px solid #f1f5f9' }}>
                                        <td style={{ padding: '10px', fontWeight: 600, color: '#0f172a', fontFamily: 'monospace' }}>
                                            {ep.path}
                                        </td>
                                        <td style={{ padding: '10px', color: '#334155' }}>
                                            <span style={{ fontWeight: 700 }}>{ep.request_count}</span> reqs
                                        </td>
                                        <td style={{ padding: '10px', color: '#64748b' }}>
                                            {ep.avg_duration_ms} ms
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}
                </div>

                {/* Top Consumers */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        <IconConsumers width={18} height={18} style={{ color: '#7c3aed' }} />
                        <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: 0 }}>Top Consumers by Ingress (24h)</h2>
                    </div>

                    {topUsers.length === 0 ? (
                        <div style={{ textAlign: 'center', padding: '40px 0', color: '#94a3b8', fontSize: '13px' }}>
                            No consumer requests recorded yet.
                        </div>
                    ) : (
                        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
                            <thead>
                                <tr style={{ borderBottom: '1px solid #e2e8f0', color: '#64748b' }}>
                                    <th style={{ padding: '8px 10px' }}>Consumer / Client</th>
                                    <th style={{ padding: '8px 10px' }}>Requests</th>
                                    <th style={{ padding: '8px 10px' }}>Type</th>
                                </tr>
                            </thead>
                            <tbody>
                                {topUsers.map((u: any, idx: number) => (
                                    <tr key={idx} style={{ borderBottom: '1px solid #f1f5f9' }}>
                                        <td style={{ padding: '10px', fontWeight: 600, color: '#0f172a' }}>
                                            {u.username}
                                        </td>
                                        <td style={{ padding: '10px', color: '#334155' }}>
                                            <span style={{ fontWeight: 700 }}>{u.request_count}</span> reqs
                                        </td>
                                        <td style={{ padding: '10px' }}>
                                            <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '10px', background: '#eff6ff', color: '#2563eb', fontWeight: 600 }}>
                                                {u.user_type || 'Consumer'}
                                            </span>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}
                </div>
            </div>
        </div>
    );
}

function PercentileCard({ title, value, label, highlight }: { title: string; value: string; label: string; highlight?: string }) {
    return (
        <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
            <span style={{ fontSize: '13px', fontWeight: 500, color: '#64748b', display: 'block', marginBottom: '8px' }}>
                {title}
            </span>
            <div style={{ fontSize: '28px', fontWeight: 800, color: highlight || '#0f172a', letterSpacing: '-0.5px', marginBottom: '4px' }}>
                {value}
            </div>
            <div style={{ fontSize: '11.5px', color: '#94a3b8' }}>{label}</div>
        </div>
    );
}
