import { useState, useEffect } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import { apiClient } from '../../api/apiClient';
import { useToast } from '../../components/Toast';
import { IconArrowLeft, IconCheck } from '../../components/Icons';

export default function UserFormPage() {
    const { id } = useParams<{ id: string }>();
    const isEdit = Boolean(id);
    const navigate = useNavigate();
    const toast = useToast();

    const [isLoading, setIsLoading] = useState(isEdit);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [showPassword, setShowPassword] = useState(false);
    const [groups, setGroups] = useState<{ id: string; name: string }[]>([]);

    const [form, setForm] = useState({
        email: '',
        password: '',
        role: 'DEVELOPER' as 'SUPER_ADMIN' | 'ADMIN' | 'DEVELOPER' | 'MAINTAINER',
        groupId: '',
    });

    useEffect(() => {
        const fetchGroups = async () => {
            try {
                const res = await apiClient.get<any>('/groups');
                const list = Array.isArray(res) ? res : (Array.isArray(res?.data) ? res.data : []);
                setGroups(list);
            } catch {
                // Non-fatal if groups cannot be loaded
            }
        };
        fetchGroups();
    }, []);

    useEffect(() => {
        if (isEdit && id) {
            loadUser(id);
        }
    }, [id, isEdit]);

    const loadUser = async (userId: string) => {
        setIsLoading(true);
        try {
            const res = await apiClient.get<any>('/users');
            const list = Array.isArray(res) ? res : (Array.isArray(res?.data) ? res.data : []);
            const target = list.find((u: any) => u.id === userId);
            if (target) {
                setForm({
                    email: target.email || '',
                    password: '',
                    role: target.role || 'DEVELOPER',
                    groupId: target.group_id || '',
                });
            } else {
                toast.error('User not found');
                navigate('/settings/users');
            }
        } catch (err: any) {
            toast.error(err.message || 'Failed to load user details');
            navigate('/settings/users');
        } finally {
            setIsLoading(false);
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!form.email.trim()) {
            toast.error('Email address is required');
            return;
        }

        if (!isEdit && !form.password) {
            toast.error('Password is required for new users');
            return;
        }

        setIsSubmitting(true);
        try {
            if (isEdit && id) {
                await apiClient.put(`/users/${id}`, {
                    email: form.email.trim(),
                    role: form.role,
                    password: form.password || undefined,
                    group_id: form.groupId || '',
                });
                toast.success('User updated successfully');
            } else {
                await apiClient.post('/users', {
                    email: form.email.trim(),
                    password: form.password,
                    role: form.role,
                    group_id: form.groupId || undefined,
                });
                toast.success('User account created successfully');
            }
            navigate('/settings/users');
        } catch (err: any) {
            toast.error(err.message || 'Failed to save user account');
        } finally {
            setIsSubmitting(false);
        }
    };

    if (isLoading) {
        return (
            <div className="overview-page-wrapper" style={{ padding: '60px 0', textAlign: 'center' }}>
                <div className="spinner" style={{ margin: '0 auto' }}></div>
                <p style={{ marginTop: '16px', color: '#64748b' }}>Loading user account details...</p>
            </div>
        );
    }

    return (
        <div className="overview-page-wrapper" style={{ paddingBottom: '60px' }}>
            {/* Breadcrumb Header */}
            <div className="overview-breadcrumb">
                <span>Settings</span>
                <span className="breadcrumb-sep">/</span>
                <Link to="/settings/users" style={{ color: '#2563eb', textDecoration: 'none' }}>
                    Users &amp; Accounts
                </Link>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">{isEdit ? 'Edit User' : 'New User'}</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">{isEdit ? 'Edit Console User' : 'New Console User'}</h1>
                    <p className="overview-subtitle">
                        Manage platform administrators, gateway operators, developers, and maintainer credentials for dashboard administration.
                    </p>
                </div>
                <div className="overview-header-actions">
                    <Link to="/settings/users" className="btn-header-refresh" style={{ textDecoration: 'none' }}>
                        <IconArrowLeft width={14} height={14} />
                        <span>Back to Users</span>
                    </Link>
                </div>
            </div>

            <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '20px', width: '100%' }}>
                {/* Card 1: Account Identity & Credentials */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Account Credentials
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                        <div className="form-group">
                            <label className="label">Email Address *</label>
                            <input
                                type="email"
                                className="input"
                                placeholder="e.g. engineer@organization.com"
                                required
                                value={form.email}
                                onChange={e => setForm(prev => ({ ...prev, email: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Used for console login and system notification dispatches.
                            </span>
                        </div>

                        <div className="form-group">
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                <label className="label">
                                    Password {isEdit ? '(Leave blank to keep unchanged)' : '*'}
                                </label>
                                <button
                                    type="button"
                                    onClick={() => setShowPassword(!showPassword)}
                                    style={{ background: 'none', border: 'none', color: '#2563eb', fontSize: '11.5px', cursor: 'pointer', padding: 0 }}
                                >
                                    {showPassword ? 'Hide' : 'Show'}
                                </button>
                            </div>
                            <input
                                type={showPassword ? 'text' : 'password'}
                                className="input"
                                placeholder={isEdit ? "•••••••• (Unchanged)" : "Enter secure login password"}
                                value={form.password}
                                onChange={e => setForm(prev => ({ ...prev, password: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Minimum 8 characters. Hashed securely with bcrypt.
                            </span>
                        </div>
                    </div>
                </div>

                {/* Card 2: Role & Access Scope */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Role &amp; Permissions Governance
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '16px', marginBottom: '20px' }}>
                        <div className="form-group">
                            <label className="label">Assigned Role *</label>
                            <select
                                className="select"
                                value={form.role}
                                onChange={e => setForm(prev => ({ ...prev, role: e.target.value as any }))}
                            >
                                <option value="DEVELOPER">Developer (API Builder &amp; SQL Queries)</option>
                                <option value="MAINTAINER">Maintainer (SRE, Upstreams &amp; TLS)</option>
                                <option value="ADMIN">Admin (Gateway &amp; Services Manager)</option>
                                <option value="SUPER_ADMIN">Super Admin (Full Root System Privileges)</option>
                            </select>
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Dictates accessible sections and operational capabilities.
                            </span>
                        </div>

                        <div className="form-group">
                            <label className="label">Tenant Group (Boundary Isolation)</label>
                            <select
                                className="select"
                                value={form.groupId}
                                onChange={e => setForm(prev => ({ ...prev, groupId: e.target.value }))}
                            >
                                <option value="">Global / Unassigned (No Tenant Boundary)</option>
                                {groups.map(g => (
                                    <option key={g.id} value={g.id}>
                                        {g.name}
                                    </option>
                                ))}
                            </select>
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Associates user with a specific tenant group boundary.
                            </span>
                        </div>

                        <div className="form-group">
                            <label className="label">Privilege Level</label>
                            <input
                                type="text"
                                className="input"
                                disabled
                                value={
                                    form.role === 'SUPER_ADMIN'
                                        ? 'Tier 1 - Full Root System Administrator'
                                        : form.role === 'ADMIN'
                                            ? 'Tier 2 - Gateway Operations & Service Configurator'
                                            : form.role === 'DEVELOPER'
                                                ? 'Tier 3 - API Builder & SQL Developer'
                                                : 'Tier 3 - SRE / DevOps & Infrastructure Maintainer'
                                }
                                style={{ background: '#f8fafc', color: '#475569', cursor: 'default' }}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Automatically mapped RBAC security classification.
                            </span>
                        </div>
                    </div>

                    {/* Visual Role Privilege Cards */}
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '12px' }}>
                        {[
                            {
                                id: 'SUPER_ADMIN',
                                title: 'Super Admin',
                                color: '#ef4444',
                                bg: '#fef2f2',
                                border: '#fecaca',
                                desc: 'Full root privileges: manages users, security keys, audit logs, and all gateway entities.'
                            },
                            {
                                id: 'ADMIN',
                                title: 'Admin',
                                color: '#2563eb',
                                bg: '#eff6ff',
                                border: '#bfdbfe',
                                desc: 'Operational control: manages routes, services, upstreams, consumers, and database instant APIs.'
                            },
                            {
                                id: 'DEVELOPER',
                                title: 'Developer',
                                color: '#16a34a',
                                bg: '#f0fdf4',
                                border: '#bbf7d0',
                                desc: 'API Builder: creates routes, instant APIs, SQL queries & preview, and tests client keys.'
                            },
                            {
                                id: 'MAINTAINER',
                                title: 'Maintainer',
                                color: '#d97706',
                                bg: '#fffbeb',
                                border: '#fde68a',
                                desc: 'SRE & DevOps: manages upstreams, targets, health checks, domain certificates, and traffic telemetry.'
                            }
                        ].map(r => {
                            const isSelected = form.role === r.id;
                            return (
                                <div
                                    key={r.id}
                                    onClick={() => setForm(prev => ({ ...prev, role: r.id as any }))}
                                    style={{
                                        border: `2px solid ${isSelected ? r.color : '#e2e8f0'}`,
                                        background: isSelected ? r.bg : '#ffffff',
                                        borderRadius: '10px',
                                        padding: '16px',
                                        cursor: 'pointer',
                                        transition: 'all 0.15s ease'
                                    }}
                                >
                                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
                                        <span style={{ fontSize: '14px', fontWeight: 700, color: isSelected ? r.color : '#1e293b' }}>
                                            {r.title}
                                        </span>
                                        {isSelected && (
                                            <div style={{ width: '18px', height: '18px', borderRadius: '50%', background: r.color, color: '#ffffff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                                <IconCheck width={12} height={12} />
                                            </div>
                                        )}
                                    </div>
                                    <p style={{ fontSize: '12px', color: '#64748b', margin: 0, lineHeight: 1.4 }}>
                                        {r.desc}
                                    </p>
                                </div>
                            );
                        })}
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
                        onClick={() => navigate('/settings/users')}
                        disabled={isSubmitting}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="btn-header-new-route"
                        disabled={isSubmitting}
                    >
                        {isSubmitting ? 'Saving...' : isEdit ? 'Update User' : 'Create User'}
                    </button>
                </div>
            </form>
        </div>
    );
}
