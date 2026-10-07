import React, { useState } from 'react';
import { useAuth } from '../../auth/AuthProvider';
import { useToast } from '../../components/Toast';
import { apiClient } from '../../api/apiClient';
import { IconKey, IconEye } from '../../components/Icons';

export default function Profile() {
    const { user } = useAuth();
    const toast = useToast();

    // Password change state
    const [currentPassword, setCurrentPassword] = useState('');
    const [newPassword, setNewPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [showCurrent, setShowCurrent] = useState(false);
    const [showNew, setShowNew] = useState(false);
    const [isUpdating, setIsUpdating] = useState(false);

    const handlePasswordChange = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!currentPassword) {
            toast.error('Current password is required');
            return;
        }

        if (newPassword.length < 8) {
            toast.error('New password must be at least 8 characters long');
            return;
        }

        if (newPassword !== confirmPassword) {
            toast.error('New password and confirmation do not match');
            return;
        }

        setIsUpdating(true);
        try {
            await apiClient.post('/auth/change-password', {
                current_password: currentPassword,
                new_password: newPassword,
            });
            toast.success('Password updated successfully!');
            setCurrentPassword('');
            setNewPassword('');
            setConfirmPassword('');
        } catch (err: any) {
            toast.error(err?.message || 'Failed to update password');
        } finally {
            setIsUpdating(false);
        }
    };

    const getInitials = (email: string) => {
        return (email || 'A').substring(0, 2).toUpperCase();
    };

    const getRoleBadgeStyle = (role: string) => {
        switch (role) {
            case 'SUPER_ADMIN':
                return { bg: '#fef2f2', color: '#b91c1c', border: '#fecaca', label: 'Super Admin' };
            case 'ADMIN':
                return { bg: '#eff6ff', color: '#1d4ed8', border: '#bfdbfe', label: 'Admin' };
            case 'DEVELOPER':
                return { bg: '#f0fdf4', color: '#15803d', border: '#bbf7d0', label: 'Developer' };
            case 'MAINTAINER':
                return { bg: '#fffbeb', color: '#b45309', border: '#fde68a', label: 'Maintainer' };
            default:
                return { bg: '#f8fafc', color: '#475569', border: '#e2e8f0', label: role };
        }
    };

    const roleBadge = getRoleBadgeStyle(user?.role || '');

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Settings</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Profile &amp; Password</span>
            </div>

            {/* Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Profile &amp; Password Settings</h1>
                    <p className="overview-subtitle">
                        Manage your administrator account details, security credentials, and update password.
                    </p>
                </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.3fr', gap: '24px', alignItems: 'start' }}>
                {/* Left Card: Account Profile Details */}
                <div style={{
                    backgroundColor: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    padding: '24px',
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px', marginBottom: '24px', paddingBottom: '20px', borderBottom: '1px solid #f1f5f9' }}>
                        <div style={{
                            width: '56px',
                            height: '56px',
                            borderRadius: '50%',
                            backgroundColor: '#eff6ff',
                            color: '#2563eb',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: '20px',
                            fontWeight: 700,
                            letterSpacing: '0.5px'
                        }}>
                            {getInitials(user?.email || '')}
                        </div>
                        <div>
                            <h2 style={{ fontSize: '18px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                                {user?.email}
                            </h2>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginTop: '6px' }}>
                                <span style={{
                                    display: 'inline-block',
                                    fontSize: '11.5px',
                                    fontWeight: 700,
                                    padding: '4px 10px',
                                    borderRadius: '6px',
                                    backgroundColor: roleBadge.bg,
                                    color: roleBadge.color,
                                    border: `1px solid ${roleBadge.border}`
                                }}>
                                    {roleBadge.label}
                                </span>
                                <span style={{
                                    fontSize: '11.5px',
                                    fontWeight: 600,
                                    padding: '4px 10px',
                                    borderRadius: '6px',
                                    backgroundColor: '#dcfce7',
                                    color: '#16a34a',
                                    border: '1px solid #bbf7d0'
                                }}>
                                    Active Session
                                </span>
                            </div>
                        </div>
                    </div>

                    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                        <div>
                            <label style={{ fontSize: '12px', fontWeight: 600, color: '#64748b', textTransform: 'uppercase' }}>
                                Administrator Email
                            </label>
                            <input
                                type="text"
                                className="input"
                                value={user?.email || ''}
                                disabled
                                style={{ backgroundColor: '#f8fafc', color: '#334155', cursor: 'not-allowed', marginTop: '4px' }}
                            />
                        </div>

                        <div>
                            <label style={{ fontSize: '12px', fontWeight: 600, color: '#64748b', textTransform: 'uppercase' }}>
                                Account User ID
                            </label>
                            <input
                                type="text"
                                className="input"
                                value={user?.id || ''}
                                disabled
                                style={{ backgroundColor: '#f8fafc', color: '#64748b', fontFamily: 'monospace', fontSize: '12px', cursor: 'not-allowed', marginTop: '4px' }}
                            />
                        </div>
                    </div>
                </div>

                {/* Right Card: Update Password */}
                <div style={{
                    backgroundColor: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    padding: '24px',
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '16px' }}>
                        <div style={{
                            width: '36px',
                            height: '36px',
                            borderRadius: '8px',
                            backgroundColor: '#eff6ff',
                            color: '#2563eb',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center'
                        }}>
                            <IconKey width={18} height={18} />
                        </div>
                        <div>
                            <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                                Update Password
                            </h2>
                            <p style={{ fontSize: '12.5px', color: '#64748b', margin: '2px 0 0 0' }}>
                                Enter your current password and choose a secure new password.
                            </p>
                        </div>
                    </div>

                    <form onSubmit={handlePasswordChange}>
                        {/* Current Password */}
                        <div className="form-group" style={{ marginBottom: '14px' }}>
                            <label className="label">Current Password</label>
                            <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
                                <input
                                    type={showCurrent ? 'text' : 'password'}
                                    className="input"
                                    value={currentPassword}
                                    onChange={e => setCurrentPassword(e.target.value)}
                                    placeholder="Enter existing password"
                                    required
                                    style={{ paddingRight: '40px' }}
                                />
                                <button
                                    type="button"
                                    onClick={() => setShowCurrent(!showCurrent)}
                                    style={{
                                        position: 'absolute',
                                        right: '10px',
                                        background: 'none',
                                        border: 'none',
                                        color: '#94a3b8',
                                        cursor: 'pointer',
                                        display: 'flex',
                                        alignItems: 'center'
                                    }}
                                >
                                    <IconEye width={16} height={16} />
                                </button>
                            </div>
                        </div>

                        {/* New Password */}
                        <div className="form-group" style={{ marginBottom: '14px' }}>
                            <label className="label">New Password</label>
                            <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
                                <input
                                    type={showNew ? 'text' : 'password'}
                                    className="input"
                                    value={newPassword}
                                    onChange={e => setNewPassword(e.target.value)}
                                    placeholder="Enter new password (min. 8 characters)"
                                    required
                                    style={{ paddingRight: '40px' }}
                                />
                                <button
                                    type="button"
                                    onClick={() => setShowNew(!showNew)}
                                    style={{
                                        position: 'absolute',
                                        right: '10px',
                                        background: 'none',
                                        border: 'none',
                                        color: '#94a3b8',
                                        cursor: 'pointer',
                                        display: 'flex',
                                        alignItems: 'center'
                                    }}
                                >
                                    <IconEye width={16} height={16} />
                                </button>
                            </div>
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Must be at least 8 characters with a mix of letters, numbers, and symbols.
                            </span>
                        </div>

                        {/* Confirm New Password */}
                        <div className="form-group" style={{ marginBottom: '20px' }}>
                            <label className="label">Confirm New Password</label>
                            <input
                                type={showNew ? 'text' : 'password'}
                                className="input"
                                value={confirmPassword}
                                onChange={e => setConfirmPassword(e.target.value)}
                                placeholder="Re-enter new password"
                                required
                            />
                            {confirmPassword && newPassword !== confirmPassword && (
                                <span style={{ fontSize: '11.5px', color: '#dc2626', marginTop: '4px', display: 'block' }}>
                                    Passwords do not match
                                </span>
                            )}
                        </div>

                        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                            <button
                                type="submit"
                                className="btn-header-new-route"
                                disabled={isUpdating}
                                style={{ width: 'auto', padding: '0 20px' }}
                            >
                                <IconKey width={14} height={14} />
                                <span>{isUpdating ? 'Updating Password...' : 'Save New Password'}</span>
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    );
}
