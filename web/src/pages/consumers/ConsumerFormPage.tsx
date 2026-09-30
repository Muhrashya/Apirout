import { useState, useEffect, FormEvent } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import { gatewayApi, ConsumerGroup } from '../../api/gatewayApi';
import { useToast } from '../../components/Toast';
import { IconArrowLeft, IconKey } from '../../components/Icons';

export default function ConsumerFormPage() {
    const { id } = useParams<{ id: string }>();
    const isEdit = Boolean(id);
    const navigate = useNavigate();
    const toast = useToast();

    const [isLoading, setIsLoading] = useState(isEdit);
    const [isSaving, setIsSaving] = useState(false);
    const [groups, setGroups] = useState<ConsumerGroup[]>([]);

    const [formData, setFormData] = useState({
        username: '',
        custom_id: '',
        group_id: '',
        description: '',
        status: 'active' as 'active' | 'suspended' | 'revoked',
        generate_key_now: true,
        key_name: 'Default API Key'
    });

    const [generatedKey, setGeneratedKey] = useState<string | null>(null);

    useEffect(() => {
        loadData();
    }, [id]);

    const loadData = async () => {
        try {
            const groupsRes = await gatewayApi.getConsumerGroups().catch(() => ({ data: [] }));
            setGroups(groupsRes.data || []);

            if (isEdit && id) {
                const consumerRes = await gatewayApi.getConsumer(id);
                const c: any = (consumerRes as any)?.data || consumerRes;
                if (c) {
                    setFormData({
                        username: c.username || '',
                        custom_id: c.custom_id || '',
                        group_id: c.group_id || '',
                        description: c.description || '',
                        status: c.status || 'active',
                        generate_key_now: false,
                        key_name: ''
                    });
                }
            }
        } catch (err: any) {
            toast.error(err.message || 'Failed to load consumer data');
        } finally {
            setIsLoading(false);
        }
    };

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        if (!formData.username.trim()) {
            toast.error('Username is required');
            return;
        }

        setIsSaving(true);
        try {
            if (isEdit && id) {
                await gatewayApi.updateConsumer(id, {
                    username: formData.username.trim(),
                    custom_id: formData.custom_id.trim() || undefined,
                    group_id: formData.group_id || undefined,
                    description: formData.description.trim() || undefined,
                    status: formData.status
                });
                toast.success('Consumer updated successfully');
                navigate('/consumers');
            } else {
                const res = await gatewayApi.createConsumer({
                    username: formData.username.trim(),
                    custom_id: formData.custom_id.trim() || undefined,
                    group_id: formData.group_id || undefined,
                    description: formData.description.trim() || undefined,
                    status: formData.status
                });

                const newConsumerId = (res as any)?.data?.id || (res as any)?.id;

                if (formData.generate_key_now && newConsumerId) {
                    try {
                        const credRes = await gatewayApi.createCredential(newConsumerId, {
                            name: formData.key_name || 'Primary API Key',
                            key_type: 'api-key'
                        });
                        const generated = (credRes as any)?.key_secret || (credRes as any)?.data?.key_secret || (credRes as any)?.key;
                        if (generated) {
                            setGeneratedKey(generated);
                            return; // stay on page to reveal key
                        }
                    } catch (credErr) {
                        console.error('Failed to create initial credential:', credErr);
                    }
                }

                toast.success('Consumer registered successfully');
                navigate('/consumers');
            }
        } catch (err: any) {
            toast.error(err.message || 'Failed to save consumer');
        } finally {
            setIsSaving(false);
        }
    };

    if (isLoading) {
        return (
            <div className="overview-page-wrapper" style={{ padding: '60px 0', textAlign: 'center' }}>
                <div className="spinner" style={{ margin: '0 auto' }}></div>
                <p style={{ marginTop: '16px', color: '#64748b' }}>Loading consumer...</p>
            </div>
        );
    }

    if (generatedKey) {
        return (
            <div style={{ maxWidth: '640px', margin: '40px auto', background: '#ffffff', borderRadius: '16px', border: '1px solid #e2e8f0', padding: '32px', boxShadow: '0 4px 20px rgba(0,0,0,0.06)' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '20px' }}>
                    <div style={{ width: '44px', height: '44px', borderRadius: '10px', background: '#f0fdf4', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#16a34a' }}>
                        <IconKey width={24} height={24} />
                    </div>
                    <div>
                        <h2 style={{ fontSize: '18px', fontWeight: 700, color: '#0f172a', margin: '0 0 4px' }}>API Key Generated</h2>
                        <p style={{ fontSize: '13px', color: '#64748b', margin: 0 }}>Save this key now. It will not be shown again.</p>
                    </div>
                </div>

                <div style={{ background: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '8px', padding: '16px', marginBottom: '24px' }}>
                    <div style={{ fontSize: '11px', fontWeight: 600, color: '#64748b', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: '6px' }}>
                        API Key (Bearer / X-API-Key)
                    </div>
                    <code style={{ fontSize: '14px', color: '#0f172a', fontFamily: 'monospace', wordBreak: 'break-all', display: 'block', userSelect: 'all', background: '#ffffff', padding: '8px 12px', border: '1px solid #cbd5e1', borderRadius: '6px' }}>
                        {generatedKey}
                    </code>
                </div>

                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
                    <button
                        type="button"
                        className="btn-header-new-route"
                        onClick={() => {
                            navigator.clipboard.writeText(generatedKey);
                            toast.success('Copied to clipboard');
                        }}
                    >
                        Copy to Clipboard
                    </button>
                    <button
                        type="button"
                        className="btn-header-refresh"
                        onClick={() => navigate('/consumers')}
                    >
                        Continue to Consumers
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div style={{ maxWidth: '1080px', margin: '0', paddingBottom: '60px' }}>
            {/* Breadcrumb Header */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', color: '#64748b', marginBottom: '16px' }}>
                <span>Gateway</span>
                <span>/</span>
                <Link to="/consumers" style={{ color: '#2563eb', textDecoration: 'none' }}>
                    Consumers
                </Link>
                <span>/</span>
                <span style={{ color: '#0f172a', fontWeight: 600 }}>{isEdit ? 'Edit Consumer' : 'New Consumer'}</span>
            </div>

            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
                <div>
                    <h1 style={{ fontSize: '24px', fontWeight: 700, color: '#0f172a', margin: '0 0 6px' }}>
                        {isEdit ? 'Edit Consumer' : 'New Gateway Consumer'}
                    </h1>
                    <p style={{ fontSize: '14px', color: '#64748b', margin: 0 }}>
                        Consumers represent external API clients, developers, or machine-to-machine integrations accessing Gateway routes.
                    </p>
                </div>
                <Link to="/consumers" className="btn-header-refresh" style={{ textDecoration: 'none' }}>
                    <IconArrowLeft width={14} height={14} />
                    <span>Back to Consumers</span>
                </Link>
            </div>

            <form onSubmit={handleSubmit}>
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        1. Consumer Identity
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                        <div className="form-group">
                            <label className="label">Username *</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="e.g. mobile-app-ios or partner-corp"
                                required
                                value={formData.username}
                                onChange={e => setFormData(prev => ({ ...prev, username: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Unique identifier representing this API consumer.
                            </span>
                        </div>

                        <div className="form-group">
                            <label className="label">Custom ID (Optional)</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="e.g. internal-uuid-9812"
                                value={formData.custom_id}
                                onChange={e => setFormData(prev => ({ ...prev, custom_id: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Map to your internal database ID or OAuth client ID.
                            </span>
                        </div>
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                        <div className="form-group">
                            <label className="label">Consumer Group (Rate Limit &amp; Tier Policy)</label>
                            <select
                                className="select"
                                value={formData.group_id}
                                onChange={e => setFormData(prev => ({ ...prev, group_id: e.target.value }))}
                            >
                                <option value="">-- No Group (Default Gateway Rate Limits) --</option>
                                {groups.map(g => (
                                    <option key={g.id} value={g.id}>
                                        {g.name} ({g.rate_limit_rpm} RPM)
                                    </option>
                                ))}
                            </select>
                        </div>

                        <div className="form-group">
                            <label className="label">Status</label>
                            <select
                                className="select"
                                value={formData.status}
                                onChange={e => setFormData(prev => ({ ...prev, status: e.target.value as any }))}
                            >
                                <option value="active">Active (Full API Access)</option>
                                <option value="suspended">Suspended (Blocked)</option>
                                <option value="revoked">Revoked (Decommissioned)</option>
                            </select>
                        </div>
                    </div>

                    <div className="form-group">
                        <label className="label">Description / Notes (Optional)</label>
                        <textarea
                            className="input"
                            rows={3}
                            placeholder="Add developer contact, department, or internal usage notes..."
                            value={formData.description}
                            onChange={e => setFormData(prev => ({ ...prev, description: e.target.value }))}
                        />
                    </div>
                </div>

                {!isEdit && (
                    <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', marginBottom: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                        <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                            2. Authentication Credentials
                        </h2>

                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '14px' }}>
                            <input
                                type="checkbox"
                                id="gen-key"
                                checked={formData.generate_key_now}
                                onChange={e => setFormData(prev => ({ ...prev, generate_key_now: e.target.checked }))}
                                style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                            />
                            <label htmlFor="gen-key" style={{ fontSize: '13.5px', fontWeight: 600, color: '#1e293b', cursor: 'pointer' }}>
                                Automatically generate initial API Key credential
                            </label>
                        </div>

                        {formData.generate_key_now && (
                            <div className="form-group" style={{ maxWidth: '400px', marginLeft: '24px' }}>
                                <label className="label">Key Description / Label</label>
                                <input
                                    type="text"
                                    className="input"
                                    value={formData.key_name}
                                    onChange={e => setFormData(prev => ({ ...prev, key_name: e.target.value }))}
                                    placeholder="e.g. Primary Production API Key"
                                />
                            </div>
                        )}
                    </div>
                )}

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
                        onClick={() => navigate('/consumers')}
                        disabled={isSaving}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="btn-header-new-route"
                        disabled={isSaving}
                    >
                        {isSaving ? 'Registering...' : isEdit ? 'Update Consumer' : 'Register Consumer'}
                    </button>
                </div>
            </form>
        </div>
    );
}
