import React, { useState, useEffect } from 'react';
import { gatewayApi, AuthPluginConfig } from '../api/gatewayApi';
import { useToast } from './Toast';
import { Modal } from './Modal';
import { IconKey, IconPlug, IconPlus, IconTrash } from './Icons';

export interface StagedAuthPlugin extends AuthPluginConfig {
    _stagedState?: 'new' | 'modified' | 'deleted';
}

interface AuthPluginManagerProps {
    scope: 'route' | 'service' | 'consumer' | 'global';
    entityId?: string;
    entityName?: string;
    onPluginChange?: () => void;
    stagedMode?: boolean;
    stagedPlugins?: AuthPluginConfig[];
    onStagedPluginsChange?: (plugins: AuthPluginConfig[]) => void;
}

const AVAILABLE_PLUGINS = [
    {
        name: 'key-auth',
        label: 'API Key (key-auth)',
        description: 'Verify requests using API keys passed via headers, query parameters, or Bearer auth.',
        defaultConfig: {
            key_names: ['X-API-Key', 'api_key'],
            key_in_header: true,
            key_in_query: true,
            hide_credentials: true
        }
    },
    {
        name: 'basic-auth',
        label: 'Basic Auth (basic-auth)',
        description: 'Verify requests using standard HTTP Basic authentication (Authorization: Basic base64(user:pass)).',
        defaultConfig: {
            hide_credentials: true,
            realm: 'Apirout API'
        }
    },
    {
        name: 'jwt-auth',
        label: 'JWT Token (jwt-auth)',
        description: 'Verify requests signed with JSON Web Tokens (HMAC, RSA, or ECDSA).',
        defaultConfig: {
            algorithms: ['HS256'],
            key_claim_name: 'iss',
            header_names: ['Authorization'],
            claims_to_verify: ['exp']
        }
    },
    {
        name: 'hmac-auth',
        label: 'HMAC Signature (hmac-auth)',
        description: 'Authenticate requests using cryptographic HMAC signature headers with clock skew protection.',
        defaultConfig: {
            algorithms: ['hmac-sha256'],
            clock_skew: 300,
            enforce_headers: ['date', 'request-line']
        }
    }
];

export const AuthPluginManager: React.FC<AuthPluginManagerProps> = ({
    scope,
    entityId,
    entityName: _entityName,
    onPluginChange,
    stagedMode = false,
    stagedPlugins,
    onStagedPluginsChange
}) => {
    const toast = useToast();
    const [internalPlugins, setInternalPlugins] = useState<AuthPluginConfig[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);
    const [selectedPluginName, setSelectedPluginName] = useState('key-auth');
    const [pluginConfigJson, setPluginConfigJson] = useState('{}');

    // Form state for config builder
    const [keyNamesInput, setKeyNamesInput] = useState('X-API-Key, api_key');
    const [keyInHeader, setKeyInHeader] = useState(true);
    const [keyInQuery, setKeyInQuery] = useState(true);
    const [hideCredentials, setHideCredentials] = useState(true);
    const [jwtAlgorithms, setJwtAlgorithms] = useState('HS256');
    const [jwtKeyClaim, setJwtKeyClaim] = useState('iss');
    const [jwtVerifyExp, setJwtVerifyExp] = useState(true);
    const [hmacClockSkew, setHmacClockSkew] = useState(300);
    const [hmacAlgo, setHmacAlgo] = useState('hmac-sha256');

    const isStaged = !!stagedMode;
    const currentPlugins = isStaged ? (stagedPlugins || []) : internalPlugins;

    const activePlugins = currentPlugins.filter(p => p._stagedState !== 'deleted');
    const attachedPluginNames = new Set(activePlugins.map(p => p.plugin_name));
    const isAllPluginsAttached = AVAILABLE_PLUGINS.every(p => attachedPluginNames.has(p.name));

    useEffect(() => {
        if (!isStaged && (entityId || scope === 'global')) {
            loadPlugins();
        }
    }, [scope, entityId, isStaged]);

    const loadPlugins = async () => {
        setIsLoading(true);
        try {
            const res = await gatewayApi.getAuthPlugins({
                scope: scope === 'global' ? 'global' : scope,
                entity_id: entityId
            });
            const data = (res as any)?.data || [];
            setInternalPlugins(Array.isArray(data) ? data : []);
        } catch (err: any) {
            console.error('Failed to load auth plugins', err);
        } finally {
            setIsLoading(false);
        }
    };

    const openAddModal = (pName?: string) => {
        let targetName = pName;
        if (!targetName || attachedPluginNames.has(targetName)) {
            const firstAvailable = AVAILABLE_PLUGINS.find(p => !attachedPluginNames.has(p.name));
            if (!firstAvailable) {
                toast.info('All available authentication plugins are already attached');
                return;
            }
            targetName = firstAvailable.name;
        }
        setSelectedPluginName(targetName);
        const meta = AVAILABLE_PLUGINS.find(p => p.name === targetName);
        if (meta) {
            setPluginConfigJson(JSON.stringify(meta.defaultConfig, null, 2));
            if (targetName === 'key-auth') {
                setKeyNamesInput('X-API-Key, api_key');
                setKeyInHeader(true);
                setKeyInQuery(true);
                setHideCredentials(true);
            } else if (targetName === 'jwt-auth') {
                setJwtAlgorithms('HS256');
                setJwtKeyClaim('iss');
                setJwtVerifyExp(true);
            } else if (targetName === 'hmac-auth') {
                setHmacClockSkew(300);
                setHmacAlgo('hmac-sha256');
            }
        }
        setIsAddModalOpen(true);
    };

    const handlePluginTypeSelect = (pName: string) => {
        setSelectedPluginName(pName);
        const meta = AVAILABLE_PLUGINS.find(p => p.name === pName);
        if (meta) {
            setPluginConfigJson(JSON.stringify(meta.defaultConfig, null, 2));
        }
    };

    const buildConfigObject = () => {
        if (selectedPluginName === 'key-auth') {
            return {
                key_names: keyNamesInput.split(',').map(s => s.trim()).filter(Boolean),
                key_in_header: keyInHeader,
                key_in_query: keyInQuery,
                hide_credentials: hideCredentials
            };
        } else if (selectedPluginName === 'jwt-auth') {
            const claims = [];
            if (jwtVerifyExp) claims.push('exp');
            return {
                algorithms: jwtAlgorithms.split(',').map(s => s.trim()).filter(Boolean),
                key_claim_name: jwtKeyClaim,
                header_names: ['Authorization'],
                claims_to_verify: claims
            };
        } else if (selectedPluginName === 'hmac-auth') {
            return {
                algorithms: [hmacAlgo],
                clock_skew: Number(hmacClockSkew) || 300,
                enforce_headers: ['date', 'request-line']
            };
        }
        try {
            return JSON.parse(pluginConfigJson);
        } catch {
            return {};
        }
    };

    const handleCreatePlugin = async (e?: React.FormEvent | React.MouseEvent) => {
        if (e) {
            e.preventDefault();
            e.stopPropagation();
        }
        const isDuplicate = currentPlugins.some(
            p => p.plugin_name === selectedPluginName && p._stagedState !== 'deleted'
        );
        if (isDuplicate) {
            toast.error(`Plugin '${selectedPluginName}' is already attached to this ${scope}`);
            return;
        }
        try {
            const configObj = buildConfigObject();

            if (isStaged) {
                const newPlugin: AuthPluginConfig = {
                    id: `staged-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`,
                    plugin_name: selectedPluginName,
                    scope: scope,
                    config: JSON.stringify(configObj),
                    enabled: true,
                    _stagedState: 'new',
                    created_at: new Date().toISOString(),
                    ...(scope === 'route' && entityId ? { route_id: entityId } : {}),
                    ...(scope === 'service' && entityId ? { service_id: entityId } : {}),
                    ...(scope === 'consumer' && entityId ? { consumer_id: entityId } : {})
                };
                if (onStagedPluginsChange) {
                    onStagedPluginsChange([...currentPlugins, newPlugin]);
                }
                toast.success(`Staged ${selectedPluginName} (pending save)`);
                setIsAddModalOpen(false);
                return;
            }

            const payload: Partial<AuthPluginConfig> = {
                plugin_name: selectedPluginName,
                scope: scope,
                config: JSON.stringify(configObj),
                enabled: true
            };

            if (scope === 'route' && entityId) payload.route_id = entityId;
            if (scope === 'service' && entityId) payload.service_id = entityId;
            if (scope === 'consumer' && entityId) payload.consumer_id = entityId;

            await gatewayApi.createAuthPlugin(payload);
            toast.success(`Attached ${selectedPluginName} to ${scope}`);
            setIsAddModalOpen(false);
            loadPlugins();
            if (onPluginChange) onPluginChange();
        } catch (err: any) {
            toast.error(err.message || 'Failed to attach plugin');
        }
    };

    const handleToggleEnabled = async (plugin: AuthPluginConfig) => {
        if (isStaged) {
            const updated = currentPlugins.map(p => {
                if (p.id !== plugin.id) return p;
                const nextEnabled = !p.enabled;
                const nextState: 'new' | 'modified' = p._stagedState === 'new' ? 'new' : 'modified';
                return {
                    ...p,
                    enabled: nextEnabled,
                    _stagedState: nextState
                };
            });
            if (onStagedPluginsChange) {
                onStagedPluginsChange(updated);
            }
            toast.success(`${plugin.plugin_name} set to ${!plugin.enabled ? 'Enabled' : 'Disabled'} (pending save)`);
            return;
        }

        try {
            await gatewayApi.updateAuthPlugin(plugin.id, {
                enabled: !plugin.enabled
            });
            toast.success(`${plugin.plugin_name} is now ${!plugin.enabled ? 'Enabled' : 'Disabled'}`);
            loadPlugins();
            if (onPluginChange) onPluginChange();
        } catch (err: any) {
            toast.error(err.message || 'Failed to toggle plugin state');
        }
    };

    const handleDeletePlugin = async (id: string, name: string) => {
        if (isStaged) {
            const target = currentPlugins.find(p => p.id === id);
            if (!target) return;
            if (target._stagedState === 'new') {
                const updated = currentPlugins.filter(p => p.id !== id);
                if (onStagedPluginsChange) {
                    onStagedPluginsChange(updated);
                }
                toast.success(`Discarded staged ${name} plugin`);
            } else {
                const updated = currentPlugins.map(p => {
                    if (p.id !== id) return p;
                    return { ...p, _stagedState: 'deleted' as const };
                });
                if (onStagedPluginsChange) {
                    onStagedPluginsChange(updated);
                }
                toast.success(`Marked ${name} for removal (pending save)`);
            }
            return;
        }

        if (!window.confirm(`Are you sure you want to remove the ${name} plugin from this ${scope}?`)) {
            return;
        }
        try {
            await gatewayApi.deleteAuthPlugin(id);
            toast.success(`Removed ${name} plugin`);
            loadPlugins();
            if (onPluginChange) onPluginChange();
        } catch (err: any) {
            toast.error(err.message || 'Failed to remove plugin');
        }
    };

    const handleUndoDelete = (id: string, name: string) => {
        if (!isStaged) return;
        const updated = currentPlugins.map(p => {
            if (p.id !== id) return p;
            return { ...p, _stagedState: undefined };
        });
        if (onStagedPluginsChange) {
            onStagedPluginsChange(updated);
        }
        toast.success(`Restored ${name} plugin`);
    };

    return (
        <div style={{
            backgroundColor: '#ffffff',
            border: '1px solid #e2e8f0',
            borderRadius: '12px',
            padding: '20px',
            boxShadow: '0 1px 3px rgba(0,0,0,0.04)'
        }}>
            <div style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: '16px',
                borderBottom: '1px solid #f1f5f9',
                paddingBottom: '12px'
            }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <div style={{
                        width: '32px',
                        height: '32px',
                        borderRadius: '8px',
                        backgroundColor: '#eff6ff',
                        color: '#2563eb',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                    }}>
                        <IconPlug width={18} height={18} />
                    </div>
                    <div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <h3 style={{ fontSize: '15px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                                Authentication Plugins
                            </h3>
                            {isStaged && (
                                <span style={{
                                    fontSize: '11px',
                                    color: '#0369a1',
                                    backgroundColor: '#e0f2fe',
                                    border: '1px solid #bae6fd',
                                    padding: '1px 8px',
                                    borderRadius: '12px',
                                    fontWeight: 600,
                                    letterSpacing: '0.02em'
                                }}>
                                    STAGED DRAFT MODE
                                </span>
                            )}
                        </div>
                        <p style={{ fontSize: '12px', color: '#64748b', margin: '2px 0 0 0' }}>
                            {isStaged
                                ? 'Inbound client identity verification (staged in form; applies when saved)'
                                : 'Inbound client identity verification (Apirout Gateway plugin chain)'}
                        </p>
                    </div>
                </div>

                <button
                    type="button"
                    disabled={isAllPluginsAttached}
                    className="btn-header-new-route"
                    style={{
                        fontSize: '12.5px',
                        padding: '6px 14px',
                        opacity: isAllPluginsAttached ? 0.6 : 1,
                        cursor: isAllPluginsAttached ? 'not-allowed' : 'pointer',
                        backgroundColor: isAllPluginsAttached ? '#94a3b8' : undefined,
                        borderColor: isAllPluginsAttached ? '#cbd5e1' : undefined
                    }}
                    title={isAllPluginsAttached ? 'All available authentication plugins are already attached' : 'Attach an authentication plugin'}
                    onClick={() => {
                        if (!isAllPluginsAttached) {
                            openAddModal();
                        }
                    }}
                >
                    <IconPlus width={14} height={14} />
                    <span>{isAllPluginsAttached ? 'All Plugins Attached' : 'Attach Plugin'}</span>
                </button>
            </div>

            {isLoading ? (
                <div style={{ padding: '24px', textAlign: 'center', color: '#64748b', fontSize: '13px' }}>
                    Loading auth plugins...
                </div>
            ) : currentPlugins.length === 0 ? (
                <div style={{
                    padding: '24px 16px',
                    textAlign: 'center',
                    backgroundColor: '#f8fafc',
                    borderRadius: '8px',
                    border: '1px dashed #cbd5e1'
                }}>
                    <IconKey width={24} height={24} style={{ color: '#94a3b8', margin: '0 auto 8px auto' }} />
                    <div style={{ fontSize: '13.5px', fontWeight: 600, color: '#334155' }}>
                        No Inbound Auth Plugins Attached
                    </div>
                    <p style={{ fontSize: '12px', color: '#64748b', margin: '4px 0 14px 0' }}>
                        By default, requests use the global fallback or route public setting. Attach an auth plugin to enforce API keys, JWT, Basic auth, or HMAC verification.
                    </p>
                    <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', flexWrap: 'wrap' }}>
                        {AVAILABLE_PLUGINS.filter(p => !attachedPluginNames.has(p.name)).slice(0, 3).map(p => (
                            <button
                                key={p.name}
                                type="button"
                                onClick={() => openAddModal(p.name)}
                                style={{
                                    border: '1px solid #e2e8f0',
                                    backgroundColor: '#ffffff',
                                    padding: '6px 12px',
                                    borderRadius: '6px',
                                    fontSize: '12px',
                                    fontWeight: 500,
                                    cursor: 'pointer',
                                    color: '#2563eb'
                                }}
                            >
                                + {p.label}
                            </button>
                        ))}
                    </div>
                </div>
            ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                    {/* Multi-Plugin Logical OR Informational Banner with Security Guidance */}
                    {activePlugins.length >= 2 && (
                        <div style={{
                            display: 'flex',
                            flexDirection: 'column',
                            gap: '6px',
                            padding: '12px 14px',
                            backgroundColor: '#eff6ff',
                            border: '1px solid #bfdbfe',
                            borderRadius: '8px',
                            color: '#1e40af',
                            fontSize: '12.5px',
                            lineHeight: 1.45
                        }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontWeight: 600 }}>
                                <span style={{ fontSize: '15px' }}>ℹ️</span>
                                <span>
                                    <strong>Multi-Plugin Logical OR:</strong> Requests are granted access if any attached plugin successfully authenticates the client (first match wins).
                                </span>
                            </div>
                            {activePlugins.length >= 3 && (
                                <div style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '6px',
                                    marginTop: '2px',
                                    padding: '6px 10px',
                                    backgroundColor: '#fef3c7',
                                    border: '1px solid #fde68a',
                                    borderRadius: '6px',
                                    color: '#92400e',
                                    fontSize: '11.5px',
                                    fontWeight: 500
                                }}>
                                    <span>⚠️</span>
                                    <span>
                                        <strong>Security Tip:</strong> Attaching 3 or more authentication mechanisms widens credential attack surface. Ensure all allowed methods meet your security policy.
                                    </span>
                                </div>
                            )}
                        </div>
                    )}
                    {currentPlugins.map(p => {
                        let parsedConfig: any = {};
                        try {
                            parsedConfig = typeof p.config === 'string' ? JSON.parse(p.config) : p.config;
                        } catch {
                            parsedConfig = {};
                        }

                        const isPendingDelete = p._stagedState === 'deleted';
                        const isNewStaged = p._stagedState === 'new';
                        const isModified = p._stagedState === 'modified';

                        return (
                            <div
                                key={p.id}
                                style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'space-between',
                                    padding: '12px 16px',
                                    backgroundColor: isPendingDelete
                                        ? '#fef2f2'
                                        : p.enabled
                                        ? '#ffffff'
                                        : '#f8fafc',
                                    border: isPendingDelete
                                        ? '1px dashed #f87171'
                                        : isNewStaged
                                        ? '1px solid #fcd34d'
                                        : '1px solid #e2e8f0',
                                    borderRadius: '8px',
                                    opacity: isPendingDelete ? 0.7 : p.enabled ? 1 : 0.75,
                                    transition: 'all 150ms ease'
                                }}
                            >
                                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                    <div style={{
                                        width: '36px',
                                        height: '36px',
                                        borderRadius: '8px',
                                        backgroundColor: isPendingDelete
                                            ? '#fee2e2'
                                            : p.enabled
                                            ? '#eff6ff'
                                            : '#f1f5f9',
                                        color: isPendingDelete
                                            ? '#ef4444'
                                            : p.enabled
                                            ? '#2563eb'
                                            : '#94a3b8',
                                        display: 'flex',
                                        alignItems: 'center',
                                        justifyContent: 'center'
                                    }}>
                                        <IconKey width={18} height={18} />
                                    </div>
                                    <div>
                                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                            <span style={{
                                                fontSize: '14px',
                                                fontWeight: 600,
                                                color: isPendingDelete ? '#991b1b' : '#0f172a',
                                                textDecoration: isPendingDelete ? 'line-through' : 'none'
                                            }}>
                                                {p.plugin_name}
                                            </span>

                                            {/* Status Badge */}
                                            {isPendingDelete ? (
                                                <span style={{
                                                    fontSize: '10.5px',
                                                    padding: '2px 7px',
                                                    borderRadius: '4px',
                                                    backgroundColor: '#fee2e2',
                                                    color: '#b91c1c',
                                                    fontWeight: 700,
                                                    border: '1px solid #fecaca'
                                                }}>
                                                    PENDING REMOVAL
                                                </span>
                                            ) : isNewStaged ? (
                                                <span style={{
                                                    fontSize: '10.5px',
                                                    padding: '2px 7px',
                                                    borderRadius: '4px',
                                                    backgroundColor: '#fef3c7',
                                                    color: '#b45309',
                                                    fontWeight: 700,
                                                    border: '1px solid #fde68a'
                                                }}>
                                                    PENDING SAVE / DRAFT
                                                </span>
                                            ) : isModified ? (
                                                <span style={{
                                                    fontSize: '10.5px',
                                                    padding: '2px 7px',
                                                    borderRadius: '4px',
                                                    backgroundColor: '#eff6ff',
                                                    color: '#1d4ed8',
                                                    fontWeight: 700,
                                                    border: '1px solid #bfdbfe'
                                                }}>
                                                    MODIFIED (PENDING SAVE)
                                                </span>
                                            ) : (
                                                <span style={{
                                                    fontSize: '10.5px',
                                                    padding: '1px 6px',
                                                    borderRadius: '4px',
                                                    backgroundColor: p.enabled ? '#dcfce7' : '#fee2e2',
                                                    color: p.enabled ? '#166534' : '#991b1b',
                                                    fontWeight: 600
                                                }}>
                                                    {p.enabled ? 'ACTIVE' : 'DISABLED'}
                                                </span>
                                            )}

                                            <span style={{
                                                fontSize: '10.5px',
                                                padding: '1px 6px',
                                                borderRadius: '4px',
                                                backgroundColor: '#f1f5f9',
                                                color: '#475569',
                                                textTransform: 'uppercase'
                                            }}>
                                                {p.scope}
                                            </span>
                                        </div>
                                        <div style={{
                                            fontSize: '12px',
                                            color: isPendingDelete ? '#94a3b8' : '#64748b',
                                            marginTop: '3px',
                                            textDecoration: isPendingDelete ? 'line-through' : 'none'
                                        }}>
                                            {p.plugin_name === 'key-auth' && (
                                                <span>Keys: {Array.isArray(parsedConfig?.key_names) ? parsedConfig.key_names.join(', ') : 'X-API-Key'}</span>
                                            )}
                                            {p.plugin_name === 'jwt-auth' && (
                                                <span>Algos: {Array.isArray(parsedConfig?.algorithms) ? parsedConfig.algorithms.join(', ') : 'HS256'} (iss claim: {parsedConfig?.key_claim_name || 'iss'})</span>
                                            )}
                                            {p.plugin_name === 'basic-auth' && (
                                                <span>Realm: {parsedConfig?.realm || 'Apirout API'}</span>
                                            )}
                                            {p.plugin_name === 'hmac-auth' && (
                                                <span>Clock Skew: {parsedConfig?.clock_skew || 300}s</span>
                                            )}

                                        </div>
                                    </div>
                                </div>

                                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                    {isPendingDelete ? (
                                        <button
                                            type="button"
                                            onClick={() => handleUndoDelete(p.id, p.plugin_name)}
                                            style={{
                                                border: '1px solid #93c5fd',
                                                backgroundColor: '#eff6ff',
                                                padding: '5px 12px',
                                                borderRadius: '6px',
                                                fontSize: '12px',
                                                cursor: 'pointer',
                                                color: '#1d4ed8',
                                                fontWeight: 600
                                            }}
                                        >
                                            Undo Removal
                                        </button>
                                    ) : (
                                        <>
                                            <button
                                                type="button"
                                                onClick={() => handleToggleEnabled(p)}
                                                style={{
                                                    border: '1px solid #e2e8f0',
                                                    backgroundColor: '#ffffff',
                                                    padding: '5px 10px',
                                                    borderRadius: '6px',
                                                    fontSize: '12px',
                                                    cursor: 'pointer',
                                                    color: p.enabled ? '#d97706' : '#16a34a',
                                                    fontWeight: 500
                                                }}
                                            >
                                                {p.enabled ? 'Disable' : 'Enable'}
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => handleDeletePlugin(p.id, p.plugin_name)}
                                                title="Remove Plugin"
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
                                                <IconTrash width={14} height={14} />
                                            </button>
                                        </>
                                    )}
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}

            {/* MODAL: ATTACH AUTH PLUGIN */}
            {isAddModalOpen && (
                <Modal
                    isOpen={isAddModalOpen}
                    onClose={() => setIsAddModalOpen(false)}
                    title={`Attach Inbound Auth Plugin to ${scope.toUpperCase()}`}
                >
                    <form onSubmit={(e) => { e.preventDefault(); e.stopPropagation(); handleCreatePlugin(e); }}>
                        <div style={{ marginBottom: '16px' }}>
                            <label className="label">Plugin Type</label>
                            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '10px', marginTop: '6px' }}>
                                {AVAILABLE_PLUGINS.map(p => {
                                    const isAttached = currentPlugins.some(
                                        existing => existing.plugin_name === p.name && existing._stagedState !== 'deleted'
                                    );
                                    const isSel = selectedPluginName === p.name;
                                    return (
                                        <div
                                            key={p.name}
                                            onClick={() => {
                                                if (!isAttached) handlePluginTypeSelect(p.name);
                                            }}
                                            style={{
                                                border: isAttached
                                                    ? '1px dashed #cbd5e1'
                                                    : isSel
                                                    ? '2px solid #2563eb'
                                                    : '2px solid #e2e8f0',
                                                backgroundColor: isAttached
                                                    ? '#f8fafc'
                                                    : isSel
                                                    ? '#eff6ff'
                                                    : '#ffffff',
                                                opacity: isAttached ? 0.6 : 1,
                                                borderRadius: '8px',
                                                padding: '10px',
                                                cursor: isAttached ? 'not-allowed' : 'pointer',
                                                textAlign: 'center',
                                                transition: 'all 150ms ease'
                                            }}
                                        >
                                            <div style={{ fontWeight: 600, fontSize: '13px', color: isAttached ? '#64748b' : isSel ? '#1d4ed8' : '#1e293b' }}>
                                                {p.name}
                                            </div>
                                            <div style={{ fontSize: '11px', color: '#64748b', marginTop: '2px' }}>
                                                {p.label.split(' ')[0]}
                                            </div>
                                            {isAttached && (
                                                <div style={{
                                                    fontSize: '9.5px',
                                                    color: '#b91c1c',
                                                    backgroundColor: '#fee2e2',
                                                    borderRadius: '4px',
                                                    padding: '1px 4px',
                                                    marginTop: '4px',
                                                    display: 'inline-block',
                                                    fontWeight: 600
                                                }}>
                                                    Already Attached
                                                </div>
                                            )}
                                        </div>
                                    );
                                })}
                            </div>
                        </div>

                        {/* CONFIG BUILDER FOR SELECTED PLUGIN */}
                        {selectedPluginName === 'key-auth' && (
                            <div style={{ backgroundColor: '#f8fafc', padding: '14px', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '16px' }}>
                                <div className="form-group" style={{ marginBottom: '10px' }}>
                                    <label className="label">API Key Param/Header Names (comma-separated)</label>
                                    <input
                                        type="text"
                                        className="input"
                                        value={keyNamesInput}
                                        onChange={e => setKeyNamesInput(e.target.value)}
                                        placeholder="X-API-Key, api_key"
                                    />
                                </div>
                                <div style={{ display: 'flex', gap: '16px' }}>
                                    <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12.5px', cursor: 'pointer' }}>
                                        <input
                                            type="checkbox"
                                            checked={keyInHeader}
                                            onChange={e => setKeyInHeader(e.target.checked)}
                                        />
                                        Check Header
                                    </label>
                                    <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12.5px', cursor: 'pointer' }}>
                                        <input
                                            type="checkbox"
                                            checked={keyInQuery}
                                            onChange={e => setKeyInQuery(e.target.checked)}
                                        />
                                        Check Query
                                    </label>
                                    <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12.5px', cursor: 'pointer' }}>
                                        <input
                                            type="checkbox"
                                            checked={hideCredentials}
                                            onChange={e => setHideCredentials(e.target.checked)}
                                        />
                                        Hide Credentials from Upstream
                                    </label>
                                </div>
                            </div>
                        )}

                        {selectedPluginName === 'jwt-auth' && (
                            <div style={{ backgroundColor: '#f8fafc', padding: '14px', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '16px' }}>
                                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '10px' }}>
                                    <div className="form-group">
                                        <label className="label">Allowed Algorithms</label>
                                        <input
                                            type="text"
                                            className="input"
                                            value={jwtAlgorithms}
                                            onChange={e => setJwtAlgorithms(e.target.value)}
                                            placeholder="HS256, RS256"
                                        />
                                    </div>
                                    <div className="form-group">
                                        <label className="label">Key Claim Name (Consumer Match)</label>
                                        <input
                                            type="text"
                                            className="input"
                                            value={jwtKeyClaim}
                                            onChange={e => setJwtKeyClaim(e.target.value)}
                                            placeholder="iss"
                                        />
                                    </div>
                                </div>
                                <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12.5px', cursor: 'pointer' }}>
                                    <input
                                        type="checkbox"
                                        checked={jwtVerifyExp}
                                        onChange={e => setJwtVerifyExp(e.target.checked)}
                                    />
                                    Verify Expiration Claim (exp)
                                </label>
                            </div>
                        )}

                        {selectedPluginName === 'hmac-auth' && (
                            <div style={{ backgroundColor: '#f8fafc', padding: '14px', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '16px' }}>
                                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
                                    <div className="form-group">
                                        <label className="label">Algorithm</label>
                                        <select
                                            className="input"
                                            value={hmacAlgo}
                                            onChange={e => setHmacAlgo(e.target.value)}
                                        >
                                            <option value="hmac-sha256">hmac-sha256</option>
                                            <option value="hmac-sha384">hmac-sha384</option>
                                            <option value="hmac-sha512">hmac-sha512</option>
                                        </select>
                                    </div>
                                    <div className="form-group">
                                        <label className="label">Clock Skew (seconds)</label>
                                        <input
                                            type="number"
                                            className="input"
                                            value={hmacClockSkew}
                                            onChange={e => setHmacClockSkew(Number(e.target.value))}
                                        />
                                    </div>
                                </div>
                            </div>
                        )}



                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
                            <button
                                type="button"
                                className="btn-header-refresh"
                                onClick={() => setIsAddModalOpen(false)}
                            >
                                Cancel
                            </button>
                            {(() => {
                                const isSelectedAttached = currentPlugins.some(
                                    p => p.plugin_name === selectedPluginName && p._stagedState !== 'deleted'
                                );
                                return (
                                    <button
                                        type="button"
                                        disabled={isSelectedAttached || !selectedPluginName}
                                        className="btn-header-new-route"
                                        style={{
                                            opacity: isSelectedAttached ? 0.5 : 1,
                                            cursor: isSelectedAttached ? 'not-allowed' : 'pointer',
                                            backgroundColor: isSelectedAttached ? '#94a3b8' : undefined,
                                            borderColor: isSelectedAttached ? '#cbd5e1' : undefined
                                        }}
                                        title={isSelectedAttached ? `${selectedPluginName} is already attached` : undefined}
                                        onClick={(e) => {
                                            e.preventDefault();
                                            e.stopPropagation();
                                            if (!isSelectedAttached) {
                                                handleCreatePlugin(e);
                                            }
                                        }}
                                    >
                                        {isStaged ? 'Stage Plugin' : 'Attach Plugin'}
                                    </button>
                                );
                            })()}
                        </div>
                    </form>
                </Modal>
            )}
        </div>
    );
};
export default AuthPluginManager;
