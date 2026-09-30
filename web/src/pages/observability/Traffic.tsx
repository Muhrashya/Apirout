import { useState, useEffect } from 'react';
import { gatewayApi, Upstream } from '../../api/gatewayApi';
import { useToast } from '../../components/Toast';
import { IconRefresh } from '../../components/Icons';

export default function TrafficPage() {
    const [stats, setStats] = useState<any>(null);
    const [upstreams, setUpstreams] = useState<Upstream[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const toast = useToast();

    useEffect(() => {
        loadTrafficData();
    }, []);

    const loadTrafficData = async () => {
        setIsLoading(true);
        try {
            const [statsRes, upstreamsRes] = await Promise.all([
                gatewayApi.getDashboardStats().catch(() => ({})),
                gatewayApi.getUpstreams().catch(() => ({ data: [] }))
            ]);

            setStats(statsRes || {});
            setUpstreams(upstreamsRes.data || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load traffic vitals');
        } finally {
            setIsLoading(false);
        }
    };

    const totalRequests = stats?.total_requests || 0;
    const success2xx = stats?.status_2xx || 0;
    const clientError4xx = stats?.status_4xx || 0;
    const serverError5xx = stats?.status_5xx || 0;
    const avgDuration = stats?.avg_duration_ms || 0;
    const uptimeSec = stats?.uptime_seconds || 0;

    const successRate = totalRequests > 0 ? ((success2xx / totalRequests) * 100).toFixed(1) : '100.0';
    const trafficTrend = stats?.traffic_trend || [];

    const formatUptime = (secs: number) => {
        const hours = Math.floor(secs / 3600);
        const mins = Math.floor((secs % 3600) / 60);
        return `${hours}h ${mins}m`;
    };

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            {/* Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', color: '#64748b', marginBottom: '4px' }}>
                        <span>Observability</span>
                        <span>/</span>
                        <span style={{ color: '#0f172a', fontWeight: 600 }}>Traffic</span>
                    </div>
                    <h1 style={{ fontSize: '24px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                        Live Gateway Traffic &amp; Vitals
                    </h1>
                </div>

                <button className="btn-header-refresh" onClick={loadTrafficData} disabled={isLoading}>
                    <IconRefresh width={14} height={14} />
                    <span>Refresh</span>
                </button>
            </div>

            {/* 4 Metric Cards */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
                <MetricCard
                    title="Total Requests (24h)"
                    value={totalRequests.toLocaleString()}
                    subtitle="Gateway ingress traffic"
                    badge={`${successRate}% Success`}
                    badgeColor="#16a34a"
                    badgeBg="#dcfce7"
                />
                <MetricCard
                    title="Average Latency"
                    value={`${avgDuration} ms`}
                    subtitle="End-to-end round trip"
                    badge="Healthy"
                    badgeColor="#2563eb"
                    badgeBg="#eff6ff"
                />
                <MetricCard
                    title="Gateway Uptime"
                    value={formatUptime(uptimeSec)}
                    subtitle="Continuous availability"
                    badge="Active"
                    badgeColor="#16a34a"
                    badgeBg="#dcfce7"
                />
                <MetricCard
                    title="Upstream Target Pools"
                    value={String(upstreams.length)}
                    subtitle="Active backend pools"
                    badge="Monitored"
                    badgeColor="#7c3aed"
                    badgeBg="#f3e8ff"
                />
            </div>

            {/* Status Code Breakdown & Upstream Health */}
            <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '20px' }}>
                {/* Traffic Distribution */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Traffic Distribution &amp; Status Code Ratio
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px', marginBottom: '24px' }}>
                        <div style={{ padding: '16px', background: '#f0fdf4', border: '1px solid #bbf7d0', borderRadius: '10px' }}>
                            <div style={{ fontSize: '12px', fontWeight: 600, color: '#16a34a', textTransform: 'uppercase' }}>2xx Success</div>
                            <div style={{ fontSize: '24px', fontWeight: 800, color: '#15803d', margin: '4px 0' }}>{success2xx}</div>
                            <div style={{ fontSize: '12px', color: '#166534' }}>Normal operations</div>
                        </div>

                        <div style={{ padding: '16px', background: '#fffbeb', border: '1px solid #fef3c7', borderRadius: '10px' }}>
                            <div style={{ fontSize: '12px', fontWeight: 600, color: '#d97706', textTransform: 'uppercase' }}>4xx Client Error</div>
                            <div style={{ fontSize: '24px', fontWeight: 800, color: '#b45309', margin: '4px 0' }}>{clientError4xx}</div>
                            <div style={{ fontSize: '12px', color: '#92400e' }}>Auth &amp; bad requests</div>
                        </div>

                        <div style={{ padding: '16px', background: '#fef2f2', border: '1px solid #fecaca', borderRadius: '10px' }}>
                            <div style={{ fontSize: '12px', fontWeight: 600, color: '#dc2626', textTransform: 'uppercase' }}>5xx Server Error</div>
                            <div style={{ fontSize: '24px', fontWeight: 800, color: '#b91c1c', margin: '4px 0' }}>{serverError5xx}</div>
                            <div style={{ fontSize: '12px', color: '#991b1b' }}>Upstream timeouts</div>
                        </div>
                    </div>

                    {/* Timeline visualization */}
                    <div style={{ borderTop: '1px solid #f1f5f9', paddingTop: '16px' }}>
                        <div style={{ fontSize: '13px', fontWeight: 600, color: '#475569', marginBottom: '12px' }}>
                            Hourly Traffic Pattern (Last 24 Hours)
                        </div>
                        {trafficTrend.length > 0 ? (
                            <div style={{ display: 'flex', alignItems: 'flex-end', height: '120px', gap: '8px', padding: '10px 0' }}>
                                {trafficTrend.map((t: any, idx: number) => {
                                    const maxVal = Math.max(...trafficTrend.map((x: any) => x.request_count || 1), 10);
                                    const heightPct = Math.max(10, Math.min(100, Math.round(((t.request_count || 0) / maxVal) * 100)));
                                    return (
                                        <div key={idx} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', height: '100%', justifyContent: 'flex-end' }}>
                                            <div
                                                title={`${t.time}: ${t.request_count} requests`}
                                                style={{
                                                    width: '100%',
                                                    height: `${heightPct}%`,
                                                    backgroundColor: '#3b82f6',
                                                    borderRadius: '4px 4px 0 0',
                                                    transition: 'all 200ms ease'
                                                }}
                                            />
                                            <span style={{ fontSize: '10px', color: '#94a3b8', marginTop: '6px' }}>{t.time}</span>
                                        </div>
                                    );
                                })}
                            </div>
                        ) : (
                            <div style={{ textAlign: 'center', padding: '30px', color: '#94a3b8', fontSize: '13px' }}>
                                Live traffic samples will appear as requests arrive.
                            </div>
                        )}
                    </div>
                </div>

                {/* Upstream Health Status */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Upstream Pool Vitals
                    </h2>

                    {upstreams.length === 0 ? (
                        <div style={{ textAlign: 'center', padding: '40px 0', color: '#94a3b8', fontSize: '13px' }}>
                            No upstream pools configured.
                        </div>
                    ) : (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                            {upstreams.map(u => (
                                <div key={u.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px', background: '#f8fafc', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
                                    <div>
                                        <div style={{ fontWeight: 600, fontSize: '13.5px', color: '#0f172a' }}>{u.name}</div>
                                        <div style={{ fontSize: '11.5px', color: '#64748b' }}>{u.algorithm} • {u.targets_count || 0} targets</div>
                                    </div>
                                    <span style={{ fontSize: '11px', fontWeight: 700, padding: '3px 8px', borderRadius: '12px', background: '#dcfce7', color: '#16a34a' }}>
                                        Healthy
                                    </span>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

function MetricCard({ title, value, subtitle, badge, badgeColor, badgeBg }: { title: string; value: string; subtitle: string; badge: string; badgeColor: string; badgeBg: string }) {
    return (
        <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                <span style={{ fontSize: '13px', fontWeight: 500, color: '#64748b' }}>{title}</span>
                <span style={{ fontSize: '11px', fontWeight: 700, padding: '2px 8px', borderRadius: '10px', background: badgeBg, color: badgeColor }}>
                    {badge}
                </span>
            </div>
            <div style={{ fontSize: '26px', fontWeight: 800, color: '#0f172a', letterSpacing: '-0.5px', marginBottom: '4px' }}>
                {value}
            </div>
            <div style={{ fontSize: '12px', color: '#94a3b8' }}>{subtitle}</div>
        </div>
    );
}
