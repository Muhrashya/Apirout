import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { apiClient } from '../../api/apiClient';
import { useToast } from '../../components/Toast';
import { IconArrowLeft } from '../../components/Icons';

export default function DomainFormPage() {
    const navigate = useNavigate();
    const toast = useToast();

    const [isSubmitting, setIsSubmitting] = useState(false);
    const [form, setForm] = useState({
        domain_name: '',
        display_name: '',
        environment: 'production',
        is_default: false,
        default_allowed_ips: '*',
        description: '',
    });

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!form.domain_name.trim()) {
            toast.error('Domain or hostname is required');
            return;
        }

        setIsSubmitting(true);
        try {
            await apiClient.post('/domains', {
                domain_name: form.domain_name.trim(),
                display_name: form.display_name.trim() || undefined,
                environment: form.environment,
                is_default: form.is_default,
                default_allowed_ips: form.default_allowed_ips.trim() || '*',
                description: form.description.trim() || undefined,
            });
            toast.success('Gateway domain registered successfully');
            navigate('/settings/domains');
        } catch (err: any) {
            toast.error(err.message || 'Failed to register domain');
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <div className="overview-page-wrapper" style={{ paddingBottom: '60px' }}>
            {/* Breadcrumb Header */}
            <div className="overview-breadcrumb">
                <span>Console Settings</span>
                <span className="breadcrumb-sep">/</span>
                <Link to="/settings/domains" style={{ color: '#2563eb', textDecoration: 'none' }}>
                    Gateway Domains
                </Link>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Register Domain</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Register Gateway Domain</h1>
                    <p className="overview-subtitle">
                        Configure custom hostnames and host header routing rules for Gateway traffic.
                    </p>
                </div>
                <div className="overview-header-actions">
                    <Link to="/settings/domains" className="btn-header-refresh" style={{ textDecoration: 'none' }}>
                        <IconArrowLeft width={14} height={14} />
                        <span>Back to Domains</span>
                    </Link>
                </div>
            </div>

            <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '20px', width: '100%' }}>
                {/* Card 1: Domain Routing Identity */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Domain Identity &amp; Routing
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                        <div className="form-group">
                            <label className="label">Domain / Hostname *</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="e.g. api.company.com or gateway.internal:8000"
                                required
                                value={form.domain_name}
                                onChange={e => setForm(prev => ({ ...prev, domain_name: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                The HTTP Host header matched by the gateway router for incoming requests.
                            </span>
                        </div>

                        <div className="form-group">
                            <label className="label">Display Label (Optional)</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="e.g. Production Public Gateway"
                                value={form.display_name}
                                onChange={e => setForm(prev => ({ ...prev, display_name: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Human-readable name shown in consoles, logs, and route bindings.
                            </span>
                        </div>
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                        <div className="form-group">
                            <label className="label">Target Environment</label>
                            <select
                                className="select"
                                value={form.environment}
                                onChange={e => setForm(prev => ({ ...prev, environment: e.target.value }))}
                            >
                                <option value="production">Production</option>
                                <option value="staging">Staging</option>
                                <option value="development">Development</option>
                            </select>
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Deployment tier tagging for environment-scoped gateway routing.
                            </span>
                        </div>

                        <div className="form-group">
                            <label className="label">Default Gateway Host</label>
                            <div style={{ display: 'flex', alignItems: 'center', height: '38px', gap: '8px' }}>
                                <input
                                    type="checkbox"
                                    id="is-default-domain"
                                    checked={form.is_default}
                                    onChange={e => setForm(prev => ({ ...prev, is_default: e.target.checked }))}
                                    style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                                />
                                <label htmlFor="is-default-domain" style={{ fontSize: '13px', color: '#1e293b', cursor: 'pointer', fontWeight: 500 }}>
                                    Set as Primary Default Host (Fallback for unmatched Host headers)
                                </label>
                            </div>
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Only one domain can serve as the default fallback route host.
                            </span>
                        </div>
                    </div>
                </div>

                {/* Card 2: Security & IP Whitelisting */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Security &amp; IP Whitelisting
                    </h2>

                    <div className="form-group">
                        <label className="label">Allowed Client IP CIDRs</label>
                        <input
                            type="text"
                            className="input"
                            placeholder="* (All) or 10.0.0.0/8, 192.168.1.0/24"
                            value={form.default_allowed_ips}
                            onChange={e => setForm(prev => ({ ...prev, default_allowed_ips: e.target.value }))}
                        />
                        <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                            Comma-separated CIDR blocks allowed to connect. Use * for public access.
                        </span>
                    </div>
                </div>

                {/* Card 3: Notes & Documentation */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Documentation &amp; Ownership
                    </h2>

                    <div className="form-group">
                        <label className="label">Description / Operational Notes (Optional)</label>
                        <textarea
                            className="input"
                            rows={3}
                            placeholder="Add department ownership, DNS record details, or routing notes..."
                            value={form.description}
                            onChange={e => setForm(prev => ({ ...prev, description: e.target.value }))}
                        />
                        <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                            Internal documentation notes to assist other engineers and platform operators.
                        </span>
                    </div>
                </div>

                {/* Bottom Action Bar */}
                <div style={{
                    display: 'flex',
                    justifyContent: 'flex-end',
                    gap: '12px',
                    padding: '16px 24px',
                    background: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    boxShadow: '0 2px 8px rgba(0,0,0,0.06)'
                }}>
                    <button
                        type="button"
                        className="btn-header-refresh"
                        onClick={() => navigate('/settings/domains')}
                        disabled={isSubmitting}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="btn-header-new-route"
                        disabled={isSubmitting}
                    >
                        {isSubmitting ? 'Registering...' : 'Register Domain'}
                    </button>
                </div>
            </form>
        </div>
    );
}
