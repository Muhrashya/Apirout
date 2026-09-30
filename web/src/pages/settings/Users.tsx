import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { apiClient } from '../../api/apiClient';
import { useToast } from '../../components/Toast';
import { useAuth } from '../../auth/AuthProvider';
import {
    IconUser, IconPlus, IconSearch, IconRefresh,
    IconPencil, IconTrash, IconCopy, IconCheck
} from '../../components/Icons';

interface User {
    id: string;
    email: string;
    role: 'SUPER_ADMIN' | 'ADMIN' | 'MEMBER' | string;
    created_at: string;
}

export default function Users() {
    const navigate = useNavigate();
    const { user: currentUser } = useAuth();
    const [users, setUsers] = useState<User[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedRoleFilter, setSelectedRoleFilter] = useState('ALL');
    const [copiedId, setCopiedId] = useState<string | null>(null);

    const toast = useToast();

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setIsLoading(true);
        try {
            const response = await apiClient.get<any>('/users');
            const userList = Array.isArray(response)
                ? response
                : (Array.isArray(response?.data) ? response.data : []);
            setUsers(userList);
        } catch (err: any) {
            toast.error(err?.message || 'Failed to load users');
        } finally {
            setIsLoading(false);
        }
    };

    const copyToClipboard = (text: string, id: string) => {
        navigator.clipboard.writeText(text);
        setCopiedId(id);
        toast.success('Copied to clipboard');
        setTimeout(() => setCopiedId(null), 2000);
    };

    const handleDelete = async (targetUser: User) => {
        if (targetUser.id === currentUser?.id) {
            toast.error('You cannot delete your own account');
            return;
        }

        if (!confirm(`Are you sure you want to permanently delete user "${targetUser.email}"?`)) {
            return;
        }

        try {
            await apiClient.delete(`/users/${targetUser.id}`);
            toast.success('User deleted successfully');
            loadData();
        } catch (err: any) {
            toast.error(err?.message || 'Failed to delete user');
        }
    };

    const filteredUsers = users.filter(u => {
        const matchesQuery = u.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
            u.id.toLowerCase().includes(searchQuery.toLowerCase());
        const matchesRole = selectedRoleFilter === 'ALL' || u.role === selectedRoleFilter;
        return matchesQuery && matchesRole;
    });

    const superAdminCount = users.filter(u => u.role === 'SUPER_ADMIN').length;
    const adminCount = users.filter(u => u.role === 'ADMIN').length;
    const memberCount = users.filter(u => u.role === 'MEMBER').length;

    const getRoleBadgeStyle = (role: string) => {
        switch (role) {
            case 'SUPER_ADMIN':
                return { bg: '#fef2f2', color: '#b91c1c', border: '#fecaca', label: 'Super Admin' };
            case 'ADMIN':
                return { bg: '#eff6ff', color: '#1d4ed8', border: '#bfdbfe', label: 'Admin' };
            case 'MEMBER':
                return { bg: '#f8fafc', color: '#475569', border: '#e2e8f0', label: 'Member' };
            default:
                return { bg: '#f8fafc', color: '#475569', border: '#e2e8f0', label: role };
        }
    };

    const getAvatarBg = (email: string) => {
        const colors = ['#2563eb', '#7c3aed', '#059669', '#d97706', '#dc2626', '#0891b2'];
        let hash = 0;
        for (let i = 0; i < email.length; i++) {
            hash = email.charCodeAt(i) + ((hash << 5) - hash);
        }
        return colors[Math.abs(hash) % colors.length];
    };

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Settings</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Users</span>
            </div>

            {/* Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Users &amp; Accounts</h1>
                    <p className="overview-subtitle">
                        Manage platform administrators and member credentials for console administration.
                    </p>
                </div>

                <div className="overview-header-actions">
                    <button
                        className="btn-header-refresh"
                        onClick={loadData}
                        title="Reload users"
                    >
                        <IconRefresh width={14} height={14} />
                        <span>Refresh</span>
                    </button>

                    <button
                        className="btn-header-new-route"
                        onClick={() => navigate('/settings/users/new')}
                    >
                        <IconPlus width={15} height={15} />
                        <span>New User</span>
                    </button>
                </div>
            </div>

            {/* Stat Summary Cards */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '16px', marginBottom: '24px' }}>
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '16px 20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ fontSize: '12.5px', color: '#64748b', fontWeight: 600, textTransform: 'uppercase', marginBottom: '6px' }}>Total Accounts</div>
                    <div style={{ fontSize: '26px', fontWeight: 700, color: '#0f172a' }}>{users.length}</div>
                    <span style={{ fontSize: '12px', color: '#94a3b8' }}>Registered console users</span>
                </div>

                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '16px 20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ fontSize: '12.5px', color: '#b91c1c', fontWeight: 600, textTransform: 'uppercase', marginBottom: '6px' }}>Super Admins</div>
                    <div style={{ fontSize: '26px', fontWeight: 700, color: '#0f172a' }}>{superAdminCount}</div>
                    <span style={{ fontSize: '12px', color: '#94a3b8' }}>Full root system privileges</span>
                </div>

                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '16px 20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ fontSize: '12.5px', color: '#1d4ed8', fontWeight: 600, textTransform: 'uppercase', marginBottom: '6px' }}>Admins</div>
                    <div style={{ fontSize: '26px', fontWeight: 700, color: '#0f172a' }}>{adminCount}</div>
                    <span style={{ fontSize: '12px', color: '#94a3b8' }}>Gateway & service managers</span>
                </div>

                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '16px 20px', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                    <div style={{ fontSize: '12.5px', color: '#475569', fontWeight: 600, textTransform: 'uppercase', marginBottom: '6px' }}>Members</div>
                    <div style={{ fontSize: '26px', fontWeight: 700, color: '#0f172a' }}>{memberCount}</div>
                    <span style={{ fontSize: '12px', color: '#94a3b8' }}>Standard portal viewers</span>
                </div>
            </div>

            {/* Filter and Search Bar */}
            <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', overflow: 'hidden', boxShadow: '0 1px 3px rgba(0,0,0,0.03)' }}>
                <div style={{ padding: '16px 20px', borderBottom: '1px solid #e2e8f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flex: 1, minWidth: '280px' }}>
                        {/* Search Input */}
                        <div style={{ position: 'relative', width: '100%', maxWidth: '340px' }}>
                            <span style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', color: '#94a3b8', display: 'flex' }}>
                                <IconSearch width={16} height={16} />
                            </span>
                            <input
                                type="text"
                                placeholder="Search by email or user ID..."
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                style={{
                                    width: '100%',
                                    padding: '8px 12px 8px 32px',
                                    border: '1px solid #cbd5e1',
                                    borderRadius: '8px',
                                    fontSize: '13px',
                                    outline: 'none'
                                }}
                            />
                        </div>

                        {/* Role Filter */}
                        <select
                            className="select select-filter"
                            value={selectedRoleFilter}
                            onChange={(e) => setSelectedRoleFilter(e.target.value)}
                        >
                            <option value="ALL">All Roles</option>
                            <option value="SUPER_ADMIN">Super Admins</option>
                            <option value="ADMIN">Admins</option>
                            <option value="MEMBER">Members</option>
                        </select>
                    </div>

                    <div style={{ fontSize: '13px', color: '#64748b' }}>
                        Showing <strong>{filteredUsers.length}</strong> of {users.length} users
                    </div>
                </div>

                {isLoading ? (
                    <div style={{ padding: '60px 20px', textAlign: 'center' }}>
                        <div className="spinner" style={{ margin: '0 auto 12px' }}></div>
                        <span style={{ color: '#64748b', fontSize: '13px' }}>Loading users...</span>
                    </div>
                ) : filteredUsers.length > 0 ? (
                    <div style={{ overflowX: 'auto' }}>
                        <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '13px' }}>
                            <thead>
                                <tr style={{ background: '#f8fafc', borderBottom: '1px solid #e2e8f0', color: '#475569', fontWeight: 600 }}>
                                    <th style={{ padding: '12px 20px' }}>User</th>
                                    <th style={{ padding: '12px 20px' }}>Role</th>
                                    <th style={{ padding: '12px 20px' }}>User ID</th>
                                    <th style={{ padding: '12px 20px' }}>Created Date</th>
                                    <th style={{ width: '120px', padding: '12px 20px', textAlign: 'right' }}>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredUsers.map((u) => {
                                    const badge = getRoleBadgeStyle(u.role);
                                    const initials = u.email.substring(0, 2).toUpperCase();
                                    const isSelf = u.id === currentUser?.id;

                                    return (
                                        <tr
                                            key={u.id}
                                            style={{
                                                borderBottom: '1px solid #f1f5f9',
                                                transition: 'background 0.15s ease'
                                            }}
                                        >
                                            <td style={{ padding: '14px 20px' }}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                                    <div
                                                        style={{
                                                            width: '36px',
                                                            height: '36px',
                                                            borderRadius: '50%',
                                                            backgroundColor: getAvatarBg(u.email),
                                                            color: '#ffffff',
                                                            display: 'flex',
                                                            alignItems: 'center',
                                                            justifyContent: 'center',
                                                            fontSize: '13px',
                                                            fontWeight: 700,
                                                            flexShrink: 0
                                                        }}
                                                    >
                                                        {initials}
                                                    </div>
                                                    <div>
                                                        <div style={{ fontWeight: 600, color: '#0f172a', display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                            {u.email}
                                                            {isSelf && (
                                                                <span style={{ fontSize: '10.5px', background: '#dcfce7', color: '#15803d', padding: '1px 6px', borderRadius: '4px', fontWeight: 700 }}>
                                                                    You
                                                                </span>
                                                            )}
                                                        </div>
                                                        <span style={{ fontSize: '12px', color: '#94a3b8' }}>
                                                            {u.role === 'SUPER_ADMIN' ? 'Root Administrator' : u.role === 'ADMIN' ? 'Gateway Manager' : 'Team Member'}
                                                        </span>
                                                    </div>
                                                </div>
                                            </td>

                                            <td style={{ padding: '14px 20px' }}>
                                                <span
                                                    style={{
                                                        display: 'inline-block',
                                                        padding: '4px 10px',
                                                        borderRadius: '6px',
                                                        fontSize: '11.5px',
                                                        fontWeight: 700,
                                                        background: badge.bg,
                                                        color: badge.color,
                                                        border: `1px solid ${badge.border}`
                                                    }}
                                                >
                                                    {badge.label}
                                                </span>
                                            </td>

                                            <td style={{ padding: '14px 20px' }}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                                    <code style={{ fontSize: '12px', color: '#64748b', fontFamily: 'monospace' }}>
                                                        {u.id.substring(0, 12)}...
                                                    </code>
                                                    <button
                                                        onClick={() => copyToClipboard(u.id, u.id)}
                                                        style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '2px', color: copiedId === u.id ? '#16a34a' : '#94a3b8' }}
                                                        title="Copy full User ID"
                                                    >
                                                        {copiedId === u.id ? <IconCheck width={13} height={13} /> : <IconCopy width={13} height={13} />}
                                                    </button>
                                                </div>
                                            </td>

                                            <td style={{ padding: '14px 20px', color: '#64748b' }}>
                                                {new Date(u.created_at).toLocaleDateString(undefined, {
                                                    year: 'numeric',
                                                    month: 'short',
                                                    day: 'numeric'
                                                })}
                                            </td>

                                            <td style={{ padding: '14px 20px', textAlign: 'right' }}>
                                                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '6px' }}>
                                                    <button
                                                        onClick={() => navigate(`/settings/users/${u.id}/edit`)}
                                                        style={{
                                                            display: 'inline-flex',
                                                            alignItems: 'center',
                                                            gap: '4px',
                                                            padding: '6px 10px',
                                                            background: '#ffffff',
                                                            border: '1px solid #cbd5e1',
                                                            borderRadius: '6px',
                                                            fontSize: '12px',
                                                            color: '#334155',
                                                            cursor: 'pointer'
                                                        }}
                                                        title="Edit User Role / Password"
                                                    >
                                                        <IconPencil width={13} height={13} />
                                                        <span>Edit</span>
                                                    </button>

                                                    <button
                                                        onClick={() => handleDelete(u)}
                                                        disabled={isSelf}
                                                        style={{
                                                            display: 'inline-flex',
                                                            alignItems: 'center',
                                                            padding: '6px 8px',
                                                            background: isSelf ? '#f8fafc' : '#ffffff',
                                                            border: `1px solid ${isSelf ? '#e2e8f0' : '#fecaca'}`,
                                                            borderRadius: '6px',
                                                            fontSize: '12px',
                                                            color: isSelf ? '#cbd5e1' : '#dc2626',
                                                            cursor: isSelf ? 'not-allowed' : 'pointer'
                                                        }}
                                                        title={isSelf ? 'Cannot delete current account' : 'Delete User'}
                                                    >
                                                        <IconTrash width={13} height={13} />
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                ) : (
                    <div style={{ padding: '48px 20px', textAlign: 'center', color: '#64748b' }}>
                        <IconUser width={36} height={36} style={{ margin: '0 auto 12px', color: '#cbd5e1' }} />
                        <h4 style={{ margin: '0 0 6px', fontSize: '15px', fontWeight: 600, color: '#1e293b' }}>
                            {searchQuery ? 'No matching users found' : 'No users registered yet'}
                        </h4>
                        <p style={{ margin: 0, fontSize: '13px' }}>
                            {searchQuery ? 'Try changing your search or role filter' : 'Create accounts to grant access to the system.'}
                        </p>
                    </div>
                )}
            </div>
        </div>
    );
}
