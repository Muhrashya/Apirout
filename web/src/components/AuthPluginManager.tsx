import React, { useState, useEffect } from 'react';
import { gatewayApi, AuthPluginConfig } from '../api/gatewayApi';
import { useToast } from './Toast';
import { Modal } from './Modal';
import { IconKey, IconPlug, IconPlus, IconTrash } from './Icons';

interface AuthPluginManagerProps {
    scope: 'route' | 'service' | 'consumer' | 'global';
    entityId?: string;
    entityName?: string;
    onPluginChange?: () => void;
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
    onPluginChange
}) => {
    const toast = useToast();
    const [plugins, setPlugins] = useState<AuthPluginConfig[]>([]);
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
    const [oauthScopes, setOauthScopes] = useState('read, write');

    useEffect(() => {
        if (entityId || scope === 'global') {
            loadPlugins();
        }
    }, [scope, entityId]);

    const loadPlugins = async () => {
        setIsLoading(true);
        try {
            const res = await gatewayApi.getAuthPlugins({
                scope: scope === 'global' ? 'global' : scope,
                entity_id: entityId
            });
            const data = (res as any)?.data || [];
            setPlugins(Array.isArray(data) ? data : []);
        } catch (err: any) {
            console.error('Failed to load auth plugins', err);
        } finally {
            setIsLoading(false);
        }
    };

    const openAddModal = (pName = 'key-auth') => {
        setSelectedPluginName(pName);
        const meta = AVAILABLE_PLUGINS.find(p => p.name === pName);
        if (meta) {
            setPluginConfigJson(JSON.stringify(meta.defaultConfig, null, 2));
            if (pName === 'key-auth') {
                setKeyNamesInput('X-API-Key, api_key');
                setKeyInHeader(true);
                setKeyInQuery(true);
                setHideCredentials(true);
            } else if (pName === 'jwt-auth') {
                setJwtAlgorithms('HS256');
                setJwtKeyClaim('iss');
                setJwtVerifyExp(true);
            } else if (pName === 'hmac-auth') {
                setHmacClockSkew(300);
                setHmacAlgo('hmac-sha256');
            } else if (pName === 'oauth2') {
                setOauthScopes('read, write');
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
        } else if (selectedPluginName === 'oauth2') {
            return {
                scopes: oauthScopes.split(',').map(s => s.trim()).filter(Boolean),
                mandatory_scope: false,
                token_expiration: 7200
            };
        }
        try {
            return JSON.parse(pluginConfigJson);
        } catch {
            return {};
        }
    };

    const handleCreatePlugin = async (e: React.FormEvent) => {
        e.preventDefault();
        try {
            const configObj = buildConfigObject();
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
                        <h3 style={{ fontSize: '15px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                            Authentication Plugins
                        </h3>
                        <p style={{ fontSize: '12px', color: '#64748b', margin: 0 }}>
                            Inbound client identity verification (Kong Gateway plugin chain)
                        </p>
                    </div>
                </div>

                <button
                    type="button"
                    className="btn-header-new-route"
                    style={{ fontSize: '12.5px', padding: '6px 14px' }}
                    onClick={() => openAddModal()}
                >
                    <IconPlus width={14} height={14} />
                    <span>Attach Plugin</span>
                </button>
            </div>

            {isLoading ? (
                <div style={{ padding: '24px', textAlign: 'center', color: '#64748b', fontSize: '13px' }}>
                    Loading auth plugins...
                </div>
            ) : plugins.length === 0 ? (
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
                        {AVAILABLE_PLUGINS.slice(0, 3).map(p => (
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
                    {plugins.map(p => {
                        let parsedConfig: any = {};
                        try {
                            parsedConfig = typeof p.config === 'string' ? JSON.parse(p.config) : p.config;
                        } catch {
                            parsedConfig = {};
                        }

                        return (
                            <div
                                key={p.id}
                                style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'space-between',
                                    padding: '12px 16px',
                                    backgroundColor: p.enabled ? '#ffffff' : '#f8fafc',
                                    border: '1px solid #e2e8f0',
                                    borderRadius: '8px',
                                    opacity: p.enabled ? 1 : 0.75
                                }}
                            >
                                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                    <div style={{
                                        width: '36px',
                                        height: '36px',
                                        borderRadius: '8px',
                                        backgroundColor: p.enabled ? '#eff6ff' : '#f1f5f9',
                                        color: p.enabled ? '#2563eb' : '#94a3b8',
                                        display: 'flex',
                                        alignItems: 'center',
                                        justifyContent: 'center'
                                    }}>
                                        <IconKey width={18} height={18} />
                                    </div>
                                    <div>
                                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                            <span style={{ fontSize: '14px', fontWeight: 600, color: '#0f172a' }}>
                                                {p.plugin_name}
                                            </span>
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
                                        <div style={{ fontSize: '12px', color: '#64748b', marginTop: '3px' }}>
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
                                            {p.plugin_name === 'oauth2' && (
                                                <span>Scopes: {Array.isArray(parsedConfig?.scopes) ? parsedConfig.scopes.join(', ') : 'read, write'}</span>
                                            )}
                                        </div>
                                    </div>
                                </div>

                                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
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
                    <form onSubmit={handleCreatePlugin}>
                        <div style={{ marginBottom: '16px' }}>
                            <label className="label">Plugin Type</label>
                            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '10px', marginTop: '6px' }}>
                                {AVAILABLE_PLUGINS.map(p => {
                                    const isSel = selectedPluginName === p.name;
                                    return (
                                        <div
                                            key={p.name}
                                            onClick={() => handlePluginTypeSelect(p.name)}
                                            style={{
                                                border: isSel ? '2px solid #2563eb' : '2px solid #e2e8f0',
                                                backgroundColor: isSel ? '#eff6ff' : '#ffffff',
                                                borderRadius: '8px',
                                                padding: '10px',
                                                cursor: 'pointer',
                                                textAlign: 'center',
                                                transition: 'border-color 150ms ease, background-color 150ms ease'
                                            }}
                                        >
                                            <div style={{ fontWeight: 600, fontSize: '13px', color: isSel ? '#1d4ed8' : '#1e293b' }}>
                                                {p.name}
                                            </div>
                                            <div style={{ fontSize: '11px', color: '#64748b', marginTop: '2px' }}>
                                                {p.label.split(' ')[0]}
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        </div>

                        {/* Plugin Specific Configuration Builder */}
                        {selectedPluginName === 'key-auth' && (
                            <div style={{ backgroundColor: '#f8fafc', padding: '14px', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '16px' }}>
                                <div style={{ marginBottom: '12px' }}>
                                    <label className="label">Header / Param Key Names (comma-separated)</label>
                                    <input
                                        type="text"
                                        className="input"
                                        value={keyNamesInput}
                                        onChange={e => setKeyNamesInput(e.target.value)}
                                        placeholder="X-API-Key, api_key"
                                        required
                                    />
                                </div>
                                <div style={{ display: 'flex', gap: '16px' }}>
                                    <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12.5px', cursor: 'pointer' }}>
                                        <input
                                            type="checkbox"
                                            checked={keyInHeader}
                                            onChange={e => setKeyInHeader(e.target.checked)}
                                        />
                                        <span>Check Headers</span>
                                    </label>
                                    <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12.5px', cursor: 'pointer' }}>
                                        <input
                                            type="checkbox"
                                            checked={keyInQuery}
                                            onChange={e => setKeyInQuery(e.target.checked)}
                                        />
                                        <span>Check Query Params</span>
                                    </label>
                                    <label style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12.5px', cursor: 'pointer' }}>
                                        <input
                                            type="checkbox"
                                            checked={hideCredentials}
                                            onChange={e => setHideCredentials(e.target.checked)}
                                        />
                                        <span>Hide from Upstream</span>
                                    </label>
                                </div>
                            </div>
                        )}

                        {selectedPluginName === 'jwt-auth' && (
                            <div style={{ backgroundColor: '#f8fafc', padding: '14px', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '16px' }}>
                                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px', marginBottom: '12px' }}>
                                    <div>
                                        <label className="label">Allowed Algorithms</label>
                                        <input
                                            type="text"
                                            className="input"
                                            value={jwtAlgorithms}
                                            onChange={e => setJwtAlgorithms(e.target.value)}
                                            placeholder="HS256, RS256"
                                        />
                                    </div>
                                    <div>
                                        <label className="label">Key Claim Name</label>
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
                                    <span>Verify Expiration (exp claim)</span>
                                </label>
                            </div>
                        )}

                        {selectedPluginName === 'hmac-auth' && (
                            <div style={{ backgroundColor: '#f8fafc', padding: '14px', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '16px' }}>
                                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
                                    <div>
                                        <label className="label">Algorithm</label>
                                        <select
                                            className="select"
                                            value={hmacAlgo}
                                            onChange={e => setHmacAlgo(e.target.value)}
                                        >
                                            <option value="hmac-sha256">hmac-sha256</option>
                                            <option value="hmac-sha512">hmac-sha512</option>
                                        </select>
                                    </div>
                                    <div>
                                        <label className="label">Clock Skew Tolerance (seconds)</label>
                                        <input
                                            type="number"
                                            className="input"
                                            min={10}
                                            max={3600}
                                            value={hmacClockSkew}
                                            onChange={e => setHmacClockSkew(parseInt(e.target.value) || 300)}
                                        />
                                    </div>
                                </div>
                            </div>
                        )}

                        {selectedPluginName === 'oauth2' && (
                            <div style={{ backgroundColor: '#f8fafc', padding: '14px', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '16px' }}>
                                <label className="label">Accepted Scopes (comma-separated)</label>
                                <input
                                    type="text"
                                    className="input"
                                    value={oauthScopes}
                                    onChange={e => setOauthScopes(e.target.value)}
                                    placeholder="read, write, admin"
                                />
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
                            <button
                                type="submit"
                                className="btn-header-new-route"
                            >
                                Attach Plugin
                            </button>
                        </div>
                    </form>
                </Modal>
            )}
        </div>
    );
};
export default AuthPluginManager;
