import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    gatewayApi,
    Consumer,
    ConsumerCredential,
    ConsumerGroup,
    Route
} from '../api/gatewayApi';
import { useToast } from '../components/Toast';
import { Modal } from '../components/Modal';
import { useAuth } from '../auth/AuthProvider';
import {
    IconConsumers,
    IconSearch,
    IconRefresh,
    IconPlus,
    IconPencil,
    IconTrash,
    IconKey,
    IconCopy,
    IconCheck,
    IconRoutes,
    IconLayers
} from '../components/Icons';

export default function ConsumersPage() {
    const navigate = useNavigate();
    const [activeTab, setActiveTab] = useState<'consumers' | 'groups'>('consumers');
    const [consumers, setConsumers] = useState<Consumer[]>([]);
    const [consumerGroups, setConsumerGroups] = useState<ConsumerGroup[]>([]);
    const [routes, setRoutes] = useState<Route[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    // Filters
    const [searchTerm, setSearchTerm] = useState('');
    const [statusFilter, setStatusFilter] = useState('all');
    const [groupFilter, setGroupFilter] = useState('all');

    // Consumer Group Modal
    const [isGroupModalOpen, setIsGroupModalOpen] = useState(false);
    const [editingGroup, setEditingGroup] = useState<ConsumerGroup | null>(null);
    const [groupForm, setGroupForm] = useState({
        name: '',
        description: '',
        rate_limit_rpm: 1000,
        rate_limit_burst: 100,
    });

    // Credentials Drawer / Modal
    const [activeConsumerForCreds, setActiveConsumerForCreds] = useState<Consumer | null>(null);
    const [credentialsList, setCredentialsList] = useState<ConsumerCredential[]>([]);
    const [isCredsLoading, setIsCredsLoading] = useState(false);
    const [isNewKeyModalOpen, setIsNewKeyModalOpen] = useState(false);
    const [newKeyForm, setNewKeyForm] = useState({
        name: 'New API Key',
        key_type: 'api-key',
        secret: '',
    });
    const [showCredSecret, setShowCredSecret] = useState(false);

    // One-Time Key Reveal Modal
    const [revealedKeySecret, setRevealedKeySecret] = useState<{
        consumerName: string;
        keyName: string;
        secret: string;
    } | null>(null);
    const [copiedKey, setCopiedKey] = useState(false);

    // Route Access Modal
    const [activeConsumerForRoutes, setActiveConsumerForRoutes] = useState<Consumer | null>(null);
    const [selectedRouteIds, setSelectedRouteIds] = useState<string[]>([]);
    const [isRoutesLoading, setIsRoutesLoading] = useState(false);
    const [routeSearchTerm, setRouteSearchTerm] = useState('');
    const [isSavingRoutes, setIsSavingRoutes] = useState(false);

    const inputRef = useRef<HTMLInputElement>(null);
    const { user, canManageEndpoints } = useAuth();
    const toast = useToast();

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setIsLoading(true);
        try {
            const [consumersRes, groupsRes, routesRes] = await Promise.all([
                gatewayApi.getConsumers().catch(() => ({ data: [] })),
                gatewayApi.getConsumerGroups().catch(() => ({ data: [] })),
                gatewayApi.getRoutes().catch(() => ({ data: [] })),
            ]);

            setConsumers(consumersRes.data || []);
            setConsumerGroups(groupsRes.data || []);
            setRoutes(routesRes.data || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load consumers');
        } finally {
            setIsLoading(false);
        }
    };

    // ==========================================
    // CONSUMER CRUD
    // ==========================================
    const openCreateConsumerModal = () => {
        navigate('/consumers/new');
    };

    const openEditConsumerModal = (consumer: Consumer) => {
        navigate(`/consumers/${consumer.id}/edit`);
    };

    const handleDeleteConsumer = async (consumer: Consumer) => {
        if (!confirm(`Are you sure you want to delete consumer "${consumer.username}"? All associated API keys and route permissions will be revoked permanently.`)) {
            return;
        }
        try {
            await gatewayApi.deleteConsumer(consumer.id);
            toast.success(`Consumer ${consumer.username} deleted`);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to delete consumer');
        }
    };

    // ==========================================
    // CONSUMER GROUPS CRUD
    // ==========================================
    const openCreateGroupModal = () => {
        setEditingGroup(null);
        setGroupForm({
            name: '',
            description: '',
            rate_limit_rpm: 1000,
            rate_limit_burst: 100,
        });
        setIsGroupModalOpen(true);
    };

    const openEditGroupModal = (group: ConsumerGroup) => {
        setEditingGroup(group);
        setGroupForm({
            name: group.name,
            description: group.description || '',
            rate_limit_rpm: group.rate_limit_rpm,
            rate_limit_burst: group.rate_limit_burst,
        });
        setIsGroupModalOpen(true);
    };

    const handleSaveGroup = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!groupForm.name.trim()) {
            toast.error('Group name is required');
            return;
        }

        try {
            if (editingGroup) {
                await gatewayApi.updateConsumerGroup(editingGroup.id, {
                    name: groupForm.name.trim(),
                    description: groupForm.description.trim() || undefined,
                    rate_limit_rpm: Number(groupForm.rate_limit_rpm) || 1000,
                    rate_limit_burst: Number(groupForm.rate_limit_burst) || 100,
                });
                toast.success('Consumer group updated successfully');
            } else {
                await gatewayApi.createConsumerGroup({
                    name: groupForm.name.trim(),
                    description: groupForm.description.trim() || undefined,
                    rate_limit_rpm: Number(groupForm.rate_limit_rpm) || 1000,
                    rate_limit_burst: Number(groupForm.rate_limit_burst) || 100,
                });
                toast.success('Consumer group created successfully');
            }
            setIsGroupModalOpen(false);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to save consumer group');
        }
    };

    const handleDeleteGroup = async (group: ConsumerGroup) => {
        if (!confirm(`Are you sure you want to delete consumer group "${group.name}"?`)) return;
        try {
            await gatewayApi.deleteConsumerGroup(group.id);
            toast.success(`Group ${group.name} deleted`);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to delete group');
        }
    };

    // ==========================================
    // CREDENTIALS MANAGEMENT
    // ==========================================
    const openCredentialsDrawer = async (consumer: Consumer) => {
        setActiveConsumerForCreds(consumer);
        setIsCredsLoading(true);
        try {
            const res = await gatewayApi.getCredentials(consumer.id);
            setCredentialsList(res.data || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load credentials');
        } finally {
            setIsCredsLoading(false);
        }
    };

    const handleCreateKey = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!activeConsumerForCreds) return;

        try {
            const res = await gatewayApi.createCredential(activeConsumerForCreds.id, {
                name: newKeyForm.name.trim() || 'API Key',
                key_type: newKeyForm.key_type,
                secret: newKeyForm.secret.trim() || undefined,
            });
            setIsNewKeyModalOpen(false);

            // Reload credentials list
            const updated = await gatewayApi.getCredentials(activeConsumerForCreds.id);
            setCredentialsList(updated.data || []);
            loadData(); // update count on main list

            // Show one-time secret
            setRevealedKeySecret({
                consumerName: activeConsumerForCreds.username,
                keyName: res.name,
                secret: res.key_secret,
            });
        } catch (err: any) {
            toast.error(err.message || 'Failed to create API key');
        }
    };

    const handleToggleCredential = async (cred: ConsumerCredential) => {
        if (!activeConsumerForCreds) return;
        try {
            await gatewayApi.toggleCredential(activeConsumerForCreds.id, cred.id, !cred.enabled);
            toast.success(`Key ${!cred.enabled ? 'enabled' : 'disabled'}`);
            const updated = await gatewayApi.getCredentials(activeConsumerForCreds.id);
            setCredentialsList(updated.data || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to toggle key status');
        }
    };

    const handleDeleteCredential = async (cred: ConsumerCredential) => {
        if (!activeConsumerForCreds) return;
        if (!confirm(`Are you sure you want to revoke key "${cred.name}"? This action cannot be undone and API requests using this key will immediately fail.`)) {
            return;
        }
        try {
            await gatewayApi.deleteCredential(activeConsumerForCreds.id, cred.id);
            toast.success('Key revoked successfully');
            const updated = await gatewayApi.getCredentials(activeConsumerForCreds.id);
            setCredentialsList(updated.data || []);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to revoke key');
        }
    };

    // ==========================================
    // ROUTE ACCESS MANAGEMENT
    // ==========================================
    const openRouteAccessDrawer = async (consumer: Consumer) => {
        setActiveConsumerForRoutes(consumer);
        setIsRoutesLoading(true);
        setRouteSearchTerm('');
        try {
            const res = await gatewayApi.getConsumerRoutes(consumer.id);
            setSelectedRouteIds(res.route_ids || []);
        } catch (err: any) {
            toast.error(err.message || 'Failed to load allowed routes');
        } finally {
            setIsRoutesLoading(false);
        }
    };

    const handleToggleRouteAccess = (routeId: string) => {
        setSelectedRouteIds(prev =>
            prev.includes(routeId) ? prev.filter(id => id !== routeId) : [...prev, routeId]
        );
    };

    const handleSelectAllRoutes = () => {
        setSelectedRouteIds(routes.map(r => r.id));
    };

    const handleDeselectAllRoutes = () => {
        setSelectedRouteIds([]);
    };

    const handleSaveRouteAccess = async () => {
        if (!activeConsumerForRoutes) return;
        setIsSavingRoutes(true);
        try {
            await gatewayApi.updateConsumerRoutes(activeConsumerForRoutes.id, selectedRouteIds);
            toast.success(`Updated route access for ${activeConsumerForRoutes.username} (${selectedRouteIds.length} routes granted)`);
            setActiveConsumerForRoutes(null);
            loadData();
        } catch (err: any) {
            toast.error(err.message || 'Failed to save route permissions');
        } finally {
            setIsSavingRoutes(false);
        }
    };

    const copySecretToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        setCopiedKey(true);
        setTimeout(() => setCopiedKey(false), 2500);
    };

    const canManage = user?.role === 'SUPER_ADMIN' || user?.role === 'ADMIN' || canManageEndpoints();

    // Filter Consumers
    const filteredConsumers = consumers.filter(c => {
        const matchesSearch =
            c.username.toLowerCase().includes(searchTerm.toLowerCase()) ||
            (c.custom_id && c.custom_id.toLowerCase().includes(searchTerm.toLowerCase())) ||
            (c.description && c.description.toLowerCase().includes(searchTerm.toLowerCase())) ||
            (c.group_name && c.group_name.toLowerCase().includes(searchTerm.toLowerCase()));

        const matchesStatus = statusFilter === 'all' || c.status === statusFilter;
        const matchesGroup = groupFilter === 'all' || c.group_id === groupFilter;

        return matchesSearch && matchesStatus && matchesGroup;
    });

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Gateway</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Consumers &amp; Groups</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">Consumers &amp; Authentication</h1>
                    <p className="overview-subtitle">
                        External API client management. Issue API keys, enforce consumer group rate limits, and restrict route access per consumer.
                    </p>
                </div>

                <div className="overview-header-actions">
                    <button className="btn-header-refresh" onClick={loadData} title="Reload data">
                        <IconRefresh width={14} height={14} />
                        <span>Refresh</span>
                    </button>
                    {canManage && (
                        activeTab === 'consumers' ? (
                            <button className="btn-header-new-route" onClick={openCreateConsumerModal}>
                                <IconPlus width={15} height={15} />
                                <span>New Consumer</span>
                            </button>
                        ) : (
                            <button className="btn-header-new-route" onClick={openCreateGroupModal}>
                                <IconPlus width={15} height={15} />
                                <span>New Consumer Group</span>
                            </button>
                        )
                    )}
                </div>
            </div>

            {/* Top Navigation Tabs */}
            <div style={{
                display: 'flex',
                gap: '8px',
                borderBottom: '1px solid #e2e8f0',
                marginBottom: '16px',
            }}>
                <button
                    onClick={() => setActiveTab('consumers')}
                    style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
                        padding: '10px 16px',
                        fontSize: '13.5px',
                        fontWeight: activeTab === 'consumers' ? 600 : 500,
                        color: activeTab === 'consumers' ? '#2563eb' : '#64748b',
                        borderBottom: activeTab === 'consumers' ? '2px solid #2563eb' : '2px solid transparent',
                        background: 'transparent',
                        borderTop: 'none',
                        borderLeft: 'none',
                        borderRight: 'none',
                        cursor: 'pointer',
                        transition: 'all 150ms ease',
                    }}
                >
                    <IconConsumers width={16} height={16} />
                    <span>Consumers</span>
                    <span style={{
                        fontSize: '11px',
                        padding: '1px 7px',
                        borderRadius: '10px',
                        backgroundColor: activeTab === 'consumers' ? '#eff6ff' : '#f1f5f9',
                        color: activeTab === 'consumers' ? '#2563eb' : '#64748b',
                        fontWeight: 600
                    }}>
                        {consumers.length}
                    </span>
                </button>

                <button
                    onClick={() => setActiveTab('groups')}
                    style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
                        padding: '10px 16px',
                        fontSize: '13.5px',
                        fontWeight: activeTab === 'groups' ? 600 : 500,
                        color: activeTab === 'groups' ? '#2563eb' : '#64748b',
                        borderBottom: activeTab === 'groups' ? '2px solid #2563eb' : '2px solid transparent',
                        background: 'transparent',
                        borderTop: 'none',
                        borderLeft: 'none',
                        borderRight: 'none',
                        cursor: 'pointer',
                        transition: 'all 150ms ease',
                    }}
                >
                    <IconLayers width={16} height={16} />
                    <span>Consumer Groups</span>
                    <span style={{
                        fontSize: '11px',
                        padding: '1px 7px',
                        borderRadius: '10px',
                        backgroundColor: activeTab === 'groups' ? '#eff6ff' : '#f1f5f9',
                        color: activeTab === 'groups' ? '#2563eb' : '#64748b',
                        fontWeight: 600
                    }}>
                        {consumerGroups.length}
                    </span>
                </button>
            </div>

            {/* TAB 1: CONSUMERS */}
            {activeTab === 'consumers' && (
                <>
                    {/* Search & Filters */}
                    <div style={{
                        backgroundColor: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '12px',
                        padding: '16px 20px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        flexWrap: 'wrap',
                        gap: '12px',
                        boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)',
                        marginBottom: '16px'
                    }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flex: '1 1 360px', flexWrap: 'wrap' }}>
                            <div style={{
                                display: 'flex',
                                alignItems: 'center',
                                backgroundColor: '#f8fafc',
                                border: '1px solid #e2e8f0',
                                borderRadius: '8px',
                                padding: '0 12px',
                                width: '100%',
                                maxWidth: '340px',
                                height: '38px',
                            }}>
                                <IconSearch width={15} height={15} style={{ color: '#94a3b8', marginRight: '8px', flexShrink: 0 }} />
                                <input
                                    ref={inputRef}
                                    type="text"
                                    placeholder="Search by username, custom_id, group... (Ctrl+K)"
                                    value={searchTerm}
                                    onChange={e => setSearchTerm(e.target.value)}
                                    style={{
                                        border: 'none',
                                        background: 'transparent',
                                        fontSize: '13px',
                                        outline: 'none',
                                        width: '100%',
                                        color: '#0f172a'
                                    }}
                                />
                            </div>

                            <select
                                className="select select-filter"
                                value={statusFilter}
                                onChange={e => setStatusFilter(e.target.value)}
                            >
                                <option value="all">All Statuses</option>
                                <option value="active">Active</option>
                                <option value="suspended">Suspended</option>
                                <option value="revoked">Revoked</option>
                            </select>

                            {consumerGroups.length > 0 && (
                                <select
                                    className="select select-filter"
                                    style={{ maxWidth: '180px' }}
                                    value={groupFilter}
                                    onChange={e => setGroupFilter(e.target.value)}
                                >
                                    <option value="all">All Groups</option>
                                    {consumerGroups.map(g => (
                                        <option key={g.id} value={g.id}>{g.name}</option>
                                    ))}
                                </select>
                            )}
                        </div>

                        <div style={{ fontSize: '13px', color: '#64748b' }}>
                            Showing <strong>{filteredConsumers.length}</strong> of {consumers.length} consumers
                        </div>
                    </div>

                    {/* Consumers Table */}
                    <div style={{
                        backgroundColor: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '12px',
                        overflow: 'hidden',
                        boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                    }}>
                        {isLoading ? (
                            <div style={{ padding: '60px 24px', textAlign: 'center' }}>
                                <div className="spinner" style={{ margin: '0 auto 12px auto' }}></div>
                                <p style={{ color: '#64748b', fontSize: '14px' }}>Loading consumers...</p>
                            </div>
                        ) : filteredConsumers.length === 0 ? (
                            <div style={{ padding: '60px 24px', textAlign: 'center' }}>
                                <div style={{
                                    width: '48px',
                                    height: '48px',
                                    borderRadius: '50%',
                                    backgroundColor: '#f1f5f9',
                                    color: '#94a3b8',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    margin: '0 auto 16px auto',
                                }}>
                                    <IconConsumers width={24} height={24} />
                                </div>
                                <h3 style={{ fontSize: '16px', fontWeight: 600, color: '#0f172a', marginBottom: '6px' }}>
                                    {consumers.length === 0 ? 'No API Consumers Registered' : 'No Matching Consumers'}
                                </h3>
                                <p style={{ fontSize: '13.5px', color: '#64748b', maxWidth: '420px', margin: '0 auto 20px auto' }}>
                                    {consumers.length === 0
                                        ? 'Register your first external consumer to issue API keys and control gateway access.'
                                        : 'Try modifying your search or status filters.'}
                                </p>
                                {canManage && consumers.length === 0 && (
                                    <button className="btn-header-new-route" onClick={openCreateConsumerModal}>
                                        Register Consumer
                                    </button>
                                )}
                            </div>
                        ) : (
                            <div style={{ overflowX: 'auto' }}>
                                <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '13.5px' }}>
                                    <thead>
                                        <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Consumer / Custom ID</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Consumer Group</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Credentials</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Route Access</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Status</th>
                                            <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569', textAlign: 'right' }}>Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {filteredConsumers.map((consumer, idx) => (
                                            <tr
                                                key={consumer.id}
                                                style={{
                                                    borderTop: idx > 0 ? '1px solid #f1f5f9' : 'none',
                                                    backgroundColor: idx % 2 === 0 ? '#ffffff' : '#fafafa',
                                                    transition: 'background 100ms ease'
                                                }}
                                            >
                                                {/* Username / Custom ID */}
                                                <td style={{ padding: '12px 20px' }}>
                                                    <div style={{ display: 'flex', flexDirection: 'column' }}>
                                                        <span style={{ fontWeight: 600, color: '#0f172a', fontSize: '14px' }}>
                                                            {consumer.username}
                                                        </span>
                                                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginTop: '2px' }}>
                                                            {consumer.custom_id && (
                                                                <code style={{
                                                                    fontSize: '11px',
                                                                    backgroundColor: '#f1f5f9',
                                                                    padding: '1px 6px',
                                                                    borderRadius: '4px',
                                                                    color: '#475569'
                                                                }}>
                                                                    id: {consumer.custom_id}
                                                                </code>
                                                            )}
                                                            {consumer.description && (
                                                                <span style={{ fontSize: '11.5px', color: '#64748b' }}>
                                                                    · {consumer.description}
                                                                </span>
                                                            )}
                                                        </div>
                                                    </div>
                                                </td>

                                                {/* Consumer Group */}
                                                <td style={{ padding: '12px 20px', color: '#475569' }}>
                                                    {consumer.group_name ? (
                                                        <span style={{
                                                            fontSize: '12px',
                                                            padding: '2px 8px',
                                                            borderRadius: '12px',
                                                            backgroundColor: '#eff6ff',
                                                            color: '#2563eb',
                                                            fontWeight: 600
                                                        }}>
                                                            {consumer.group_name}
                                                        </span>
                                                    ) : (
                                                        <span style={{ color: '#94a3b8', fontSize: '12.5px' }}>Default / None</span>
                                                    )}
                                                </td>

                                                {/* Credentials Count & Button */}
                                                <td style={{ padding: '12px 20px' }}>
                                                    <button
                                                        onClick={() => openCredentialsDrawer(consumer)}
                                                        style={{
                                                            display: 'inline-flex',
                                                            alignItems: 'center',
                                                            gap: '6px',
                                                            border: '1px solid #e2e8f0',
                                                            borderRadius: '6px',
                                                            padding: '4px 10px',
                                                            backgroundColor: '#ffffff',
                                                            color: '#334155',
                                                            fontSize: '12px',
                                                            fontWeight: 500,
                                                            cursor: 'pointer'
                                                        }}
                                                    >
                                                        <IconKey width={13} height={13} style={{ color: '#d97706' }} />
                                                        <span>{consumer.credentials_count || 0} API Keys</span>
                                                    </button>
                                                </td>

                                                {/* Route Access Count & Button */}
                                                <td style={{ padding: '12px 20px' }}>
                                                    <button
                                                        onClick={() => openRouteAccessDrawer(consumer)}
                                                        style={{
                                                            display: 'inline-flex',
                                                            alignItems: 'center',
                                                            gap: '6px',
                                                            border: '1px solid #e2e8f0',
                                                            borderRadius: '6px',
                                                            padding: '4px 10px',
                                                            backgroundColor: '#ffffff',
                                                            color: '#334155',
                                                            fontSize: '12px',
                                                            fontWeight: 500,
                                                            cursor: 'pointer'
                                                        }}
                                                    >
                                                        <IconRoutes width={13} height={13} style={{ color: '#2563eb' }} />
                                                        <span>{consumer.allowed_endpoints_count || 0} Routes Allowed</span>
                                                    </button>
                                                </td>

                                                {/* Status */}
                                                <td style={{ padding: '12px 20px' }}>
                                                    <span style={{
                                                        fontSize: '11px',
                                                        fontWeight: 600,
                                                        padding: '2px 8px',
                                                        borderRadius: '12px',
                                                        backgroundColor:
                                                            consumer.status === 'active' ? '#dcfce7' :
                                                            consumer.status === 'suspended' ? '#fef3c7' : '#fee2e2',
                                                        color:
                                                            consumer.status === 'active' ? '#16a34a' :
                                                            consumer.status === 'suspended' ? '#d97706' : '#dc2626',
                                                        textTransform: 'capitalize'
                                                    }}>
                                                        {consumer.status}
                                                    </span>
                                                </td>

                                                {/* Actions */}
                                                <td style={{ padding: '12px 20px', textAlign: 'right' }}>
                                                    <div style={{ display: 'inline-flex', gap: '8px' }}>
                                                        <button
                                                            onClick={() => openCredentialsDrawer(consumer)}
                                                            title="Manage API Keys"
                                                            style={{
                                                                border: '1px solid #e2e8f0',
                                                                backgroundColor: '#ffffff',
                                                                padding: '6px',
                                                                borderRadius: '6px',
                                                                color: '#d97706',
                                                                cursor: 'pointer'
                                                            }}
                                                        >
                                                            <IconKey width={14} height={14} />
                                                        </button>
                                                        <button
                                                            onClick={() => openRouteAccessDrawer(consumer)}
                                                            title="Manage Route Access"
                                                            style={{
                                                                border: '1px solid #e2e8f0',
                                                                backgroundColor: '#ffffff',
                                                                padding: '6px',
                                                                borderRadius: '6px',
                                                                color: '#2563eb',
                                                                cursor: 'pointer'
                                                            }}
                                                        >
                                                            <IconRoutes width={14} height={14} />
                                                        </button>
                                                        {canManage && (
                                                            <button
                                                                onClick={() => openEditConsumerModal(consumer)}
                                                                title="Edit Consumer"
                                                                style={{
                                                                    border: '1px solid #e2e8f0',
                                                                    backgroundColor: '#ffffff',
                                                                    padding: '6px',
                                                                    borderRadius: '6px',
                                                                    color: '#475569',
                                                                    cursor: 'pointer'
                                                                }}
                                                            >
                                                                <IconPencil width={14} height={14} />
                                                            </button>
                                                        )}
                                                        {canManage && (
                                                            <button
                                                                onClick={() => handleDeleteConsumer(consumer)}
                                                                title="Delete Consumer"
                                                                style={{
                                                                    border: '1px solid #fee2e2',
                                                                    backgroundColor: '#ffffff',
                                                                    padding: '6px',
                                                                    borderRadius: '6px',
                                                                    color: '#dc2626',
                                                                    cursor: 'pointer'
                                                                }}
                                                            >
                                                                <IconTrash width={14} height={14} />
                                                            </button>
                                                        )}
                                                    </div>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        )}
                    </div>
                </>
            )}

            {/* TAB 2: CONSUMER GROUPS */}
            {activeTab === 'groups' && (
                <div style={{
                    backgroundColor: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    overflow: 'hidden',
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                }}>
                    {isLoading ? (
                        <div style={{ padding: '60px 24px', textAlign: 'center' }}>
                            <div className="spinner" style={{ margin: '0 auto 12px auto' }}></div>
                            <p style={{ color: '#64748b', fontSize: '14px' }}>Loading consumer groups...</p>
                        </div>
                    ) : consumerGroups.length === 0 ? (
                        <div style={{ padding: '60px 24px', textAlign: 'center' }}>
                            <div style={{
                                width: '48px',
                                height: '48px',
                                borderRadius: '50%',
                                backgroundColor: '#f1f5f9',
                                color: '#94a3b8',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                margin: '0 auto 16px auto',
                            }}>
                                <IconLayers width={24} height={24} />
                            </div>
                            <h3 style={{ fontSize: '16px', fontWeight: 600, color: '#0f172a', marginBottom: '6px' }}>
                                No Consumer Groups
                            </h3>
                            <p style={{ fontSize: '13.5px', color: '#64748b', maxWidth: '420px', margin: '0 auto 20px auto' }}>
                                Consumer Groups apply shared traffic rate limits and access policies to multiple external clients.
                            </p>
                            {canManage && (
                                <button className="btn-header-new-route" onClick={openCreateGroupModal}>
                                    Create Consumer Group
                                </button>
                            )}
                        </div>
                    ) : (
                        <div style={{ overflowX: 'auto' }}>
                            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '13.5px' }}>
                                <thead>
                                    <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                                        <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Group Name / Description</th>
                                        <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Rate Limit (RPM)</th>
                                        <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Burst Allowance</th>
                                        <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569' }}>Consumers Count</th>
                                        <th style={{ padding: '12px 20px', fontWeight: 600, color: '#475569', textAlign: 'right' }}>Actions</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {consumerGroups.map((group, idx) => (
                                        <tr
                                            key={group.id}
                                            style={{
                                                borderTop: idx > 0 ? '1px solid #f1f5f9' : 'none',
                                                backgroundColor: idx % 2 === 0 ? '#ffffff' : '#fafafa',
                                                transition: 'background 100ms ease'
                                            }}
                                        >
                                            <td style={{ padding: '12px 20px' }}>
                                                <div style={{ display: 'flex', flexDirection: 'column' }}>
                                                    <span style={{ fontWeight: 600, color: '#0f172a', fontSize: '14px' }}>
                                                        {group.name}
                                                    </span>
                                                    {group.description && (
                                                        <span style={{ fontSize: '12px', color: '#64748b', marginTop: '2px' }}>
                                                            {group.description}
                                                        </span>
                                                    )}
                                                </div>
                                            </td>

                                            <td style={{ padding: '12px 20px', color: '#0f172a', fontWeight: 600 }}>
                                                {group.rate_limit_rpm} <span style={{ fontWeight: 400, color: '#64748b', fontSize: '12px' }}>req/min</span>
                                            </td>

                                            <td style={{ padding: '12px 20px', color: '#475569' }}>
                                                {group.rate_limit_burst} reqs
                                            </td>

                                            <td style={{ padding: '12px 20px' }}>
                                                <span style={{
                                                    fontSize: '12px',
                                                    padding: '2px 8px',
                                                    borderRadius: '12px',
                                                    backgroundColor: '#f1f5f9',
                                                    color: '#334155',
                                                    fontWeight: 600
                                                }}>
                                                    {group.consumers_count || 0} members
                                                </span>
                                            </td>

                                            <td style={{ padding: '12px 20px', textAlign: 'right' }}>
                                                <div style={{ display: 'inline-flex', gap: '8px' }}>
                                                    {canManage && (
                                                        <button
                                                            onClick={() => openEditGroupModal(group)}
                                                            title="Edit Group"
                                                            style={{
                                                                border: '1px solid #e2e8f0',
                                                                backgroundColor: '#ffffff',
                                                                padding: '6px',
                                                                borderRadius: '6px',
                                                                color: '#2563eb',
                                                                cursor: 'pointer'
                                                            }}
                                                        >
                                                            <IconPencil width={14} height={14} />
                                                        </button>
                                                    )}
                                                    {canManage && (
                                                        <button
                                                            onClick={() => handleDeleteGroup(group)}
                                                            title="Delete Group"
                                                            style={{
                                                                border: '1px solid #fee2e2',
                                                                backgroundColor: '#ffffff',
                                                                padding: '6px',
                                                                borderRadius: '6px',
                                                                color: '#dc2626',
                                                                cursor: 'pointer'
                                                            }}
                                                        >
                                                            <IconTrash width={14} height={14} />
                                                        </button>
                                                    )}
                                                </div>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </div>
            )}

            {/* MODAL: CREATE / EDIT CONSUMER GROUP */}
            {/* ========================================================= */}
            <Modal
                isOpen={isGroupModalOpen}
                onClose={() => setIsGroupModalOpen(false)}
                title={editingGroup ? `Edit Consumer Group: ${editingGroup.name}` : 'Create Consumer Group'}
            >
                <form onSubmit={handleSaveGroup}>
                    <div className="form-group" style={{ marginBottom: '14px' }}>
                        <label className="label">Group Name *</label>
                        <input
                            type="text"
                            className="input"
                            value={groupForm.name}
                            onChange={e => setGroupForm({ ...groupForm, name: e.target.value })}
                            placeholder="e.g. Internal Applications or Gold Partner Tier"
                            required
                        />
                    </div>

                    <div className="form-group" style={{ marginBottom: '14px' }}>
                        <label className="label">Description</label>
                        <input
                            type="text"
                            className="input"
                            value={groupForm.description}
                            onChange={e => setGroupForm({ ...groupForm, description: e.target.value })}
                            placeholder="Policy scope or SLA details"
                        />
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px', marginBottom: '16px' }}>
                        <div>
                            <label className="label">Rate Limit (RPM)</label>
                            <input
                                type="number"
                                className="input"
                                value={groupForm.rate_limit_rpm}
                                onChange={e => setGroupForm({ ...groupForm, rate_limit_rpm: parseInt(e.target.value) || 1000 })}
                                min="1"
                                required
                            />
                            <span style={{ fontSize: '11px', color: '#64748b' }}>Allowed requests per minute</span>
                        </div>

                        <div>
                            <label className="label">Burst Allowance</label>
                            <input
                                type="number"
                                className="input"
                                value={groupForm.rate_limit_burst}
                                onChange={e => setGroupForm({ ...groupForm, rate_limit_burst: parseInt(e.target.value) || 100 })}
                                min="1"
                                required
                            />
                            <span style={{ fontSize: '11px', color: '#64748b' }}>Immediate burst capacity</span>
                        </div>
                    </div>

                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
                        <button type="button" className="btn-header-refresh" onClick={() => setIsGroupModalOpen(false)}>
                            Cancel
                        </button>
                        <button type="submit" className="btn-header-new-route">
                            {editingGroup ? 'Update Group' : 'Create Group'}
                        </button>
                    </div>
                </form>
            </Modal>

            {/* ========================================================= */}
            {/* DRAWER / MODAL: CREDENTIALS (API KEYS) */}
            {/* ========================================================= */}
            {activeConsumerForCreds && (
                <Modal
                    isOpen={!!activeConsumerForCreds}
                    onClose={() => setActiveConsumerForCreds(null)}
                    title={`API Credentials: ${activeConsumerForCreds.username}`}
                >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                            <p style={{ fontSize: '13px', color: '#64748b', margin: 0 }}>
                                Secret keys are stored securely using bcrypt hashing. Plaintext secrets are only revealed once.
                            </p>
                            {canManage && (
                                <button
                                    className="btn-header-new-route"
                                    onClick={() => {
                                        setNewKeyForm({ name: 'API Key', key_type: 'api-key', secret: '' });
                                        setIsNewKeyModalOpen(true);
                                    }}
                                    style={{ fontSize: '12.5px', padding: '6px 12px' }}
                                >
                                    <IconPlus width={13} height={13} />
                                    <span>Generate Key</span>
                                </button>
                            )}
                        </div>

                        {isCredsLoading ? (
                            <div style={{ padding: '30px', textAlign: 'center' }}>
                                <div className="spinner" style={{ margin: '0 auto 8px auto' }}></div>
                                <span style={{ fontSize: '13px', color: '#64748b' }}>Loading keys...</span>
                            </div>
                        ) : credentialsList.length === 0 ? (
                            <div style={{
                                padding: '36px 16px',
                                textAlign: 'center',
                                backgroundColor: '#f8fafc',
                                borderRadius: '8px',
                                border: '1px dashed #cbd5e1'
                            }}>
                                <IconKey width={28} height={28} style={{ color: '#94a3b8', margin: '0 auto 8px auto' }} />
                                <div style={{ fontSize: '14px', fontWeight: 600, color: '#334155' }}>No Active API Keys</div>
                                <p style={{ fontSize: '12.5px', color: '#64748b', margin: '4px 0 12px 0' }}>
                                    This consumer does not have any active credentials. Generate an API key to authenticate requests.
                                </p>
                                {canManage && (
                                    <button
                                        className="btn-header-new-route"
                                        onClick={() => setIsNewKeyModalOpen(true)}
                                    >
                                        Generate First API Key
                                    </button>
                                )}
                            </div>
                        ) : (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                {credentialsList.map(cred => (
                                    <div
                                        key={cred.id}
                                        style={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            justifyContent: 'space-between',
                                            padding: '12px 14px',
                                            backgroundColor: '#ffffff',
                                            border: '1px solid #e2e8f0',
                                            borderRadius: '8px',
                                        }}
                                    >
                                        <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                                <span style={{ fontSize: '13px', fontWeight: 600, color: '#0f172a' }}>
                                                    {cred.name}
                                                </span>
                                                <span style={{
                                                    fontSize: '11px',
                                                    padding: '1px 6px',
                                                    borderRadius: '4px',
                                                    backgroundColor: cred.enabled ? '#dcfce7' : '#fee2e2',
                                                    color: cred.enabled ? '#16a34a' : '#dc2626',
                                                    fontWeight: 600
                                                }}>
                                                    {cred.enabled ? 'Active' : 'Disabled'}
                                                </span>
                                            </div>
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                                <code style={{
                                                    fontFamily: 'monospace',
                                                    fontSize: '12px',
                                                    color: '#475569',
                                                    backgroundColor: '#f1f5f9',
                                                    padding: '2px 6px',
                                                    borderRadius: '4px'
                                                }}>
                                                    {cred.masked_key || 'apirout_live_••••••••'}
                                                </code>
                                                <span style={{ fontSize: '11.5px', color: '#94a3b8' }}>
                                                    Created: {new Date(cred.created_at).toLocaleDateString()}
                                                </span>
                                            </div>
                                        </div>

                                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                            {canManage && (
                                                <button
                                                    onClick={() => handleToggleCredential(cred)}
                                                    style={{
                                                        border: '1px solid #e2e8f0',
                                                        backgroundColor: '#ffffff',
                                                        padding: '5px 10px',
                                                        borderRadius: '6px',
                                                        fontSize: '12px',
                                                        cursor: 'pointer',
                                                        color: cred.enabled ? '#d97706' : '#16a34a'
                                                    }}
                                                >
                                                    {cred.enabled ? 'Disable' : 'Enable'}
                                                </button>
                                            )}
                                            {canManage && (
                                                <button
                                                    onClick={() => handleDeleteCredential(cred)}
                                                    title="Revoke Key"
                                                    style={{
                                                        border: '1px solid #fee2e2',
                                                        backgroundColor: '#ffffff',
                                                        padding: '5px 8px',
                                                        borderRadius: '6px',
                                                        fontSize: '12px',
                                                        cursor: 'pointer',
                                                        color: '#dc2626'
                                                    }}
                                                >
                                                    <IconTrash width={13} height={13} />
                                                </button>
                                            )}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}

                        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '12px' }}>
                            <button className="btn-header-refresh" onClick={() => setActiveConsumerForCreds(null)}>
                                Close
                            </button>
                        </div>
                    </div>
                </Modal>
            )}

            {/* ========================================================= */}
            {/* SUB-MODAL: GENERATE KEY FORM */}
            {/* ========================================================= */}
            {isNewKeyModalOpen && activeConsumerForCreds && (
                <Modal
                    isOpen={isNewKeyModalOpen}
                    onClose={() => setIsNewKeyModalOpen(false)}
                    title={`Generate API Key for ${activeConsumerForCreds.username}`}
                >
                    <form onSubmit={handleCreateKey}>
                        <div className="form-group" style={{ marginBottom: '14px' }}>
                            <label className="label">Key Name / Description</label>
                            <input
                                type="text"
                                className="input"
                                value={newKeyForm.name}
                                onChange={e => setNewKeyForm({ ...newKeyForm, name: e.target.value })}
                                placeholder="e.g. Production Mobile App Key"
                                required
                            />
                        </div>

                        <div className="form-group" style={{ marginBottom: '16px' }}>
                            <label className="label">Credential Type</label>
                            <select
                                className="select"
                                value={newKeyForm.key_type}
                                onChange={e => setNewKeyForm({ ...newKeyForm, key_type: e.target.value })}
                            >
                                <option value="api-key">Standard API Key (Header X-API-Key or ?api_key)</option>
                                <option value="basic-auth">Basic Auth Password (HTTP Basic Authorization)</option>
                                <option value="jwt">JWT Token Key / Shared Secret</option>
                                <option value="hmac-auth">HMAC-SHA256 Shared Secret</option>
                                <option value="oauth2">OAuth 2.0 Client Secret</option>
                            </select>
                        </div>

                        <div className="form-group" style={{ marginBottom: '16px' }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                                <label className="label" style={{ margin: 0 }}>
                                    {newKeyForm.key_type === 'basic-auth' ? 'Consumer Password' : 'Custom Secret (Optional)'}
                                </label>
                                <button
                                    type="button"
                                    onClick={() => setShowCredSecret(!showCredSecret)}
                                    style={{ background: 'none', border: 'none', color: '#2563eb', fontSize: '12px', cursor: 'pointer', padding: 0 }}
                                >
                                    {showCredSecret ? 'Hide' : 'Show'}
                                </button>
                            </div>
                            <input
                                type={showCredSecret ? 'text' : 'password'}
                                className="input"
                                value={newKeyForm.secret}
                                onChange={e => setNewKeyForm({ ...newKeyForm, secret: e.target.value })}
                                placeholder={newKeyForm.key_type === 'basic-auth' ? 'Enter password (or leave blank to auto-generate)' : 'Leave blank to auto-generate a secure random secret'}
                            />
                            <span style={{ fontSize: '11px', color: '#64748b', marginTop: '3px', display: 'block' }}>
                                Credentials are encrypted with bcrypt at rest. The plain secret is shown only once after creation.
                            </span>
                        </div>

                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
                            <button type="button" className="btn-header-refresh" onClick={() => setIsNewKeyModalOpen(false)}>
                                Cancel
                            </button>
                            <button type="submit" className="btn-header-new-route">
                                Generate Key
                            </button>
                        </div>
                    </form>
                </Modal>
            )}

            {/* ========================================================= */}
            {/* MODAL: ONE-TIME API KEY REVEAL (CRITICAL) */}
            {/* ========================================================= */}
            {revealedKeySecret && (
                <Modal
                    isOpen={!!revealedKeySecret}
                    onClose={() => setRevealedKeySecret(null)}
                    title="⚠️ Copy Your Secret API Key"
                >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                        <div style={{
                            padding: '12px 14px',
                            backgroundColor: '#fffbeb',
                            border: '1px solid #fde68a',
                            borderRadius: '8px',
                            fontSize: '13px',
                            color: '#92400e'
                        }}>
                            <strong>Warning:</strong> Please copy this secret key now. For your security, this key is encrypted with bcrypt and will <strong>never be shown again</strong>.
                        </div>

                        <div>
                            <span style={{ fontSize: '12px', color: '#64748b' }}>
                                Consumer: <strong>{revealedKeySecret.consumerName}</strong> ({revealedKeySecret.keyName})
                            </span>
                            <div style={{
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'space-between',
                                backgroundColor: '#0f172a',
                                padding: '12px 16px',
                                borderRadius: '8px',
                                marginTop: '6px'
                            }}>
                                <code style={{
                                    color: '#38bdf8',
                                    fontFamily: 'monospace',
                                    fontSize: '13.5px',
                                    wordBreak: 'break-all'
                                }}>
                                    {revealedKeySecret.secret}
                                </code>
                                <button
                                    onClick={() => copySecretToClipboard(revealedKeySecret.secret)}
                                    style={{
                                        marginLeft: '12px',
                                        backgroundColor: copiedKey ? '#16a34a' : '#2563eb',
                                        color: '#ffffff',
                                        border: 'none',
                                        borderRadius: '6px',
                                        padding: '6px 12px',
                                        fontSize: '12.5px',
                                        fontWeight: 600,
                                        cursor: 'pointer',
                                        display: 'flex',
                                        alignItems: 'center',
                                        gap: '4px',
                                        flexShrink: 0
                                    }}
                                >
                                    {copiedKey ? <IconCheck width={14} height={14} /> : <IconCopy width={14} height={14} />}
                                    <span>{copiedKey ? 'Copied!' : 'Copy Key'}</span>
                                </button>
                            </div>
                        </div>

                        <div style={{ fontSize: '12.5px', color: '#64748b' }}>
                            Authenticate HTTP requests using the header:
                            <pre style={{
                                backgroundColor: '#f1f5f9',
                                color: '#0f172a',
                                padding: '8px 12px',
                                borderRadius: '6px',
                                marginTop: '4px',
                                fontSize: '12px'
                            }}>
                                {`X-API-Key: ${revealedKeySecret.secret}`}
                            </pre>
                        </div>

                        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '10px' }}>
                            <button
                                className="btn-header-new-route"
                                onClick={() => setRevealedKeySecret(null)}
                            >
                                Done, I Have Saved It
                            </button>
                        </div>
                    </div>
                </Modal>
            )}

            {/* ========================================================= */}
            {/* DRAWER / MODAL: ROUTE ACCESS CONTROL */}
            {/* ========================================================= */}
            {activeConsumerForRoutes && (
                <Modal
                    isOpen={!!activeConsumerForRoutes}
                    onClose={() => setActiveConsumerForRoutes(null)}
                    title={`Route Access Permissions: ${activeConsumerForRoutes.username}`}
                >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                        <p style={{ fontSize: '13px', color: '#64748b', margin: 0 }}>
                            Grant or revoke access to specific Gateway Routes for this consumer. Unchecked routes will return 403 Forbidden.
                        </p>

                        {/* Search & Bulk Select */}
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '10px' }}>
                            <div style={{
                                display: 'flex',
                                alignItems: 'center',
                                backgroundColor: '#f8fafc',
                                border: '1px solid #e2e8f0',
                                borderRadius: '6px',
                                padding: '0 10px',
                                flex: 1,
                                height: '34px'
                            }}>
                                <IconSearch width={14} height={14} style={{ color: '#94a3b8', marginRight: '6px' }} />
                                <input
                                    type="text"
                                    placeholder="Filter available routes..."
                                    value={routeSearchTerm}
                                    onChange={e => setRouteSearchTerm(e.target.value)}
                                    style={{
                                        border: 'none',
                                        background: 'transparent',
                                        fontSize: '12.5px',
                                        outline: 'none',
                                        width: '100%'
                                    }}
                                />
                            </div>

                            <div style={{ display: 'flex', gap: '6px' }}>
                                <button
                                    type="button"
                                    onClick={handleSelectAllRoutes}
                                    style={{
                                        border: '1px solid #e2e8f0',
                                        backgroundColor: '#ffffff',
                                        padding: '6px 10px',
                                        borderRadius: '6px',
                                        fontSize: '12px',
                                        cursor: 'pointer'
                                    }}
                                >
                                    Select All
                                </button>
                                <button
                                    type="button"
                                    onClick={handleDeselectAllRoutes}
                                    style={{
                                        border: '1px solid #e2e8f0',
                                        backgroundColor: '#ffffff',
                                        padding: '6px 10px',
                                        borderRadius: '6px',
                                        fontSize: '12px',
                                        cursor: 'pointer'
                                    }}
                                >
                                    Clear All
                                </button>
                            </div>
                        </div>

                        {/* Routes List */}
                        {isRoutesLoading ? (
                            <div style={{ padding: '30px', textAlign: 'center' }}>
                                <div className="spinner" style={{ margin: '0 auto 8px auto' }}></div>
                                <span style={{ fontSize: '13px', color: '#64748b' }}>Loading route permissions...</span>
                            </div>
                        ) : routes.length === 0 ? (
                            <div style={{ padding: '24px', textAlign: 'center', color: '#64748b', fontSize: '13px' }}>
                                No routes configured yet in the gateway.
                            </div>
                        ) : (
                            <div style={{
                                maxHeight: '360px',
                                overflowY: 'auto',
                                border: '1px solid #e2e8f0',
                                borderRadius: '8px'
                            }}>
                                {routes
                                    .filter(r =>
                                        r.path.toLowerCase().includes(routeSearchTerm.toLowerCase()) ||
                                        r.method.toLowerCase().includes(routeSearchTerm.toLowerCase()) ||
                                        (r.name && r.name.toLowerCase().includes(routeSearchTerm.toLowerCase()))
                                    )
                                    .map((r, i) => {
                                        const isChecked = selectedRouteIds.includes(r.id);
                                        return (
                                            <div
                                                key={r.id}
                                                onClick={() => handleToggleRouteAccess(r.id)}
                                                style={{
                                                    display: 'flex',
                                                    alignItems: 'center',
                                                    gap: '12px',
                                                    padding: '10px 14px',
                                                    borderTop: i > 0 ? '1px solid #f1f5f9' : 'none',
                                                    backgroundColor: isChecked ? '#eff6ff' : '#ffffff',
                                                    cursor: 'pointer',
                                                    transition: 'background 100ms ease'
                                                }}
                                            >
                                                <input
                                                    type="checkbox"
                                                    checked={isChecked}
                                                    onChange={() => {}} // handled by parent onClick
                                                    style={{ cursor: 'pointer' }}
                                                />
                                                <span style={{
                                                    fontSize: '11px',
                                                    fontWeight: 700,
                                                    padding: '2px 6px',
                                                    borderRadius: '4px',
                                                    fontFamily: 'monospace',
                                                    backgroundColor:
                                                        r.method === 'GET' ? '#dcfce7' :
                                                        r.method === 'POST' ? '#dbeafe' :
                                                        r.method === 'PUT' ? '#fef3c7' : '#fee2e2',
                                                    color:
                                                        r.method === 'GET' ? '#16a34a' :
                                                        r.method === 'POST' ? '#2563eb' :
                                                        r.method === 'PUT' ? '#d97706' : '#dc2626'
                                                }}>
                                                    {r.method}
                                                </span>
                                                <code style={{
                                                    fontFamily: 'monospace',
                                                    fontSize: '13px',
                                                    fontWeight: 600,
                                                    color: '#0f172a'
                                                }}>
                                                    {r.path}
                                                </code>
                                                {r.name && r.name !== r.path && (
                                                    <span style={{ fontSize: '12px', color: '#64748b' }}>
                                                        ({r.name})
                                                    </span>
                                                )}
                                                {r.is_public && (
                                                    <span style={{
                                                        marginLeft: 'auto',
                                                        fontSize: '10.5px',
                                                        padding: '1px 6px',
                                                        borderRadius: '10px',
                                                        backgroundColor: '#f1f5f9',
                                                        color: '#64748b'
                                                    }}>
                                                        Public
                                                    </span>
                                                )}
                                            </div>
                                        );
                                    })}
                            </div>
                        )}

                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: '12px' }}>
                            <span style={{ fontSize: '13px', color: '#64748b' }}>
                                <strong>{selectedRouteIds.length}</strong> of {routes.length} routes granted
                            </span>

                            <div style={{ display: 'flex', gap: '10px' }}>
                                <button
                                    type="button"
                                    className="btn-header-refresh"
                                    onClick={() => setActiveConsumerForRoutes(null)}
                                >
                                    Cancel
                                </button>
                                <button
                                    type="button"
                                    className="btn-header-new-route"
                                    onClick={handleSaveRouteAccess}
                                    disabled={isSavingRoutes}
                                >
                                    {isSavingRoutes ? 'Saving...' : 'Save Permissions'}
                                </button>
                            </div>
                        </div>
                    </div>
                </Modal>
            )}
        </div>
    );
}
