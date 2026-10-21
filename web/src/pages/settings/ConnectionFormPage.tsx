import { useState, useEffect } from 'react';
import { useNavigate, useParams, useSearchParams, Link } from 'react-router-dom';
import { gatewayApi, Connection } from '../../api/gatewayApi';
import { useToast } from '../../components/Toast';
import {
    DatabaseEnginePickerModal,
    DATABASE_ENGINES,
    DatabaseEngineOption
} from '../../components/DatabaseEnginePickerModal';
import {
    IconArrowLeft,
    IconCheck,
    IconAlertTriangle,
    IconBolt
} from '../../components/Icons';

export default function ConnectionFormPage() {
    const { id } = useParams<{ id: string }>();
    const isEdit = Boolean(id);
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const toast = useToast();

    const engineFromUrl = searchParams.get('engine') || 'postgres';

    const [isLoading, setIsLoading] = useState(isEdit);
    const [isSaving, setIsSaving] = useState(false);
    const [isTesting, setIsTesting] = useState(false);
    const [showPassword, setShowPassword] = useState(false);
    const [isPickerModalOpen, setIsPickerModalOpen] = useState(false);
    
    // Engine specific UI state
    const [mongoUriMode, setMongoUriMode] = useState(false);
    const [mongoUri, setMongoUri] = useState('');
    const [trustServerCert, setTrustServerCert] = useState(true);
    const [mssqlInstance, setMssqlInstance] = useState('');
    const [testResult, setTestResult] = useState<{ success: boolean; latency_ms?: number; error?: string } | null>(null);

    const [form, setForm] = useState({
        name: '',
        db_type: engineFromUrl,
        host: 'localhost',
        port: 5432,
        dbname: '',
        username: 'postgres',
        password: '',
        sslmode: 'disable',
        auth_source: 'admin',
        max_open_conns: 10,
        max_idle_conns: 5,
        conn_max_lifetime: 300,
        is_read_only: false
    });

    const activeEngine: DatabaseEngineOption =
        DATABASE_ENGINES.find(e => e.id === form.db_type) || DATABASE_ENGINES[0];

    useEffect(() => {
        if (isEdit && id) {
            loadConnection(id);
        } else if (!isEdit) {
            applyEngineDefaults(engineFromUrl);
        }
    }, [id, isEdit, engineFromUrl]);

    const applyEngineDefaults = (newType: string) => {
        let defaultPort = 5432;
        let defaultUser = 'postgres';
        let defaultHost = 'localhost';
        let defaultDb = '';
        let defaultAuthSource = 'admin';
        let defaultSsl = 'disable';

        if (newType === 'mysql') {
            defaultPort = 3306;
            defaultUser = 'root';
            defaultDb = 'main_db';
        } else if (newType === 'mssql') {
            defaultPort = 1433;
            defaultUser = 'sa';
            defaultDb = 'master';
            defaultSsl = 'disable';
        } else if (newType === 'mongodb' || newType === 'mongo') {
            defaultPort = 27017;
            defaultUser = 'admin';
            defaultDb = 'admin';
            defaultAuthSource = 'admin';
            defaultSsl = 'disable';
        } else if (newType === 'trino') {
            defaultPort = 8080;
            defaultUser = 'trino';
            defaultDb = 'tpch';
            defaultSsl = 'disable';
        }

        setForm(prev => ({
            ...prev,
            db_type: newType,
            port: defaultPort,
            host: prev.host === 'localhost' ? defaultHost : prev.host,
            dbname: defaultDb,
            username: defaultUser,
            auth_source: defaultAuthSource,
            sslmode: defaultSsl
        }));
    };

    const handleSelectEngineFromModal = (selectedEngineId: string) => {
        setIsPickerModalOpen(false);
        applyEngineDefaults(selectedEngineId);
        setSearchParams({ engine: selectedEngineId }, { replace: true });
    };

    const loadConnection = async (connId: string) => {
        setIsLoading(true);
        try {
            const res = await gatewayApi.getConnection(connId);
            const conn: Connection = (res as any)?.data || res;
            if (conn) {
                setForm({
                    name: conn.name || '',
                    db_type: conn.db_type || 'postgres',
                    host: conn.host || 'localhost',
                    port: conn.port || 5432,
                    dbname: conn.db_name || (conn as any).dbname || '',
                    username: conn.username || '',
                    password: '', // Kept empty for security unless changed
                    sslmode: conn.ssl_mode || (conn as any).sslmode || 'disable',
                    auth_source: 'admin',
                    max_open_conns: conn.max_open_conns || 10,
                    max_idle_conns: conn.max_idle_conns || 5,
                    conn_max_lifetime: conn.conn_max_lifetime || 300,
                    is_read_only: conn.is_read_only || false
                });
            }
        } catch (err: any) {
            toast.error(err.message || 'Failed to load database connection details');
            navigate('/settings/connections');
        } finally {
            setIsLoading(false);
        }
    };

    const handleMongoUriChange = (uri: string) => {
        setMongoUri(uri);
        if (uri.startsWith('mongodb://') || uri.startsWith('mongodb+srv://')) {
            try {
                const clean = uri.replace('mongodb+srv://', 'http://').replace('mongodb://', 'http://');
                const parsed = new URL(clean);
                setForm(prev => ({
                    ...prev,
                    host: parsed.hostname || prev.host,
                    port: parsed.port ? parseInt(parsed.port, 10) : 27017,
                    username: parsed.username ? decodeURIComponent(parsed.username) : prev.username,
                    password: parsed.password ? decodeURIComponent(parsed.password) : prev.password,
                    dbname: parsed.pathname ? parsed.pathname.replace('/', '') : prev.dbname,
                }));
            } catch {
                // Ignore transient typing parse errors
            }
        }
    };

    const handleTestConnection = async () => {
        if (!form.host.trim()) {
            toast.error('Host is required before testing.');
            return;
        }

        if (!form.dbname.trim() && !mongoUriMode) {
            toast.error('Database name is required before testing.');
            return;
        }

        setIsTesting(true);
        setTestResult(null);

        try {
            const payload: any = {
                db_type: form.db_type,
                host: (form.db_type === 'mongodb' && mongoUriMode && mongoUri) ? mongoUri : form.host.trim(),
                port: Number(form.port),
                dbname: form.dbname.trim(),
                username: form.username.trim(),
                password: form.password,
                sslmode: form.sslmode,
            };

            const res = await gatewayApi.testRawConnection(payload);
            setTestResult(res);
            if (res.success) {
                toast.success(`Connection verified! Response time: ${res.latency_ms || 1} ms`);
            } else {
                toast.error(res.error || 'Connection failed');
            }
        } catch (err: any) {
            const errMsg = err.message || 'Database connection test failed';
            setTestResult({ success: false, error: errMsg });
            toast.error(errMsg);
        } finally {
            setIsTesting(false);
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!form.name.trim()) {
            toast.error('Connection profile name is required');
            return;
        }
        if (!form.host.trim()) {
            toast.error('Host / IP address is required');
            return;
        }
        if (!form.dbname.trim() && !mongoUriMode) {
            toast.error('Database name is required');
            return;
        }

        setIsSaving(true);
        try {
            const payload: any = {
                name: form.name.trim(),
                db_type: form.db_type,
                host: (form.db_type === 'mongodb' && mongoUriMode && mongoUri) ? mongoUri : form.host.trim(),
                port: Number(form.port),
                dbname: form.dbname.trim(),
                db_name: form.dbname.trim(),
                username: form.username.trim(),
                sslmode: form.sslmode,
                ssl_mode: form.sslmode,
                max_open_conns: Number(form.max_open_conns) || 10,
                max_idle_conns: Number(form.max_idle_conns) || 5,
                conn_max_lifetime: Number(form.conn_max_lifetime) || 300,
                is_read_only: form.is_read_only
            };

            if (form.password) {
                payload.password = form.password;
            }

            if (isEdit && id) {
                await gatewayApi.updateConnection(id, payload);
                toast.success('Database connection updated successfully');
            } else {
                await gatewayApi.createConnection(payload);
                toast.success(`${activeEngine.name} source registered successfully`);
            }

            navigate('/settings/connections');
        } catch (err: any) {
            toast.error(err.message || 'Failed to save database connection');
        } finally {
            setIsSaving(false);
        }
    };

    if (isLoading) {
        return (
            <div className="overview-page-wrapper" style={{ padding: '60px 0', textAlign: 'center' }}>
                <div className="spinner" style={{ margin: '0 auto' }}></div>
                <p style={{ marginTop: '16px', color: '#64748b' }}>Loading connection details...</p>
            </div>
        );
    }

    const isMongo = form.db_type === 'mongodb' || form.db_type === 'mongo';
    const isMssql = form.db_type === 'mssql';
    const isPostgres = form.db_type === 'postgres';
    const isMysql = form.db_type === 'mysql';
    const isTrino = form.db_type === 'trino';

    return (
        <div className="overview-page-wrapper" style={{ paddingBottom: '60px' }}>
            {/* Breadcrumb Header */}
            <div className="overview-breadcrumb">
                <span>Settings</span>
                <span className="breadcrumb-sep">/</span>
                <Link to="/settings/connections" style={{ color: '#2563eb', textDecoration: 'none' }}>
                    Database Sources
                </Link>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">{isEdit ? `Edit ${activeEngine.name}` : `New ${activeEngine.name}`}</span>
            </div>

            {/* Page Header */}
            <div className="overview-header">
                <div className="overview-title-col">
                    <h1 className="overview-title">
                        {isEdit ? `Edit ${activeEngine.name} Source` : `New ${activeEngine.name} Source`}
                    </h1>
                    <p className="overview-subtitle">
                        Configure target database credentials for Instant API generation, SQL/NoSQL query runners, and data proxying.
                    </p>
                </div>
                <div className="overview-header-actions">
                    <Link to="/settings/connections" className="btn-header-refresh" style={{ textDecoration: 'none' }}>
                        <IconArrowLeft width={14} height={14} />
                        <span>Back to Connections</span>
                    </Link>
                </div>
            </div>

            <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '20px', width: '100%' }}>
                {/* DBeaver Style Engine Profile Banner */}
                <div style={{
                    background: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    padding: '18px 24px',
                    boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    flexWrap: 'wrap',
                    gap: '16px'
                }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                        <div style={{
                            width: '48px',
                            height: '48px',
                            borderRadius: '10px',
                            background: '#f8fafc',
                            border: '1px solid #f1f5f9',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            flexShrink: 0,
                            padding: '4px'
                        }}>
                            {activeEngine.icon}
                        </div>
                        <div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                                <span style={{ fontSize: '17px', fontWeight: 700, color: '#0f172a' }}>
                                    {activeEngine.name}
                                </span>
                                <span style={{ fontSize: '12px', color: '#64748b', fontWeight: 600 }}>
                                    {activeEngine.categoryLabel}
                                </span>
                            </div>
                            <p style={{ margin: '4px 0 0', fontSize: '12.5px', color: '#64748b' }}>
                                {activeEngine.description}
                            </p>
                        </div>
                    </div>

                    {!isEdit && (
                        <button
                            type="button"
                            className="btn-header-refresh"
                            onClick={() => setIsPickerModalOpen(true)}
                            style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', whiteSpace: 'nowrap' }}
                        >
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                                <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67" />
                            </svg>
                            <span>Change Database Engine</span>
                        </button>
                    )}
                </div>

                {/* Card 1: Engine & Connection Identity */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Database Source Identity
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                        <div className="form-group">
                            <label className="label">Connection Profile Name *</label>
                            <input
                                type="text"
                                className="input"
                                placeholder={`e.g. Core ${activeEngine.name} DB or Analytics Warehouse`}
                                required
                                value={form.name}
                                onChange={e => setForm(prev => ({ ...prev, name: e.target.value }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Friendly alias shown in Instant API generator and SQL/NoSQL query runners.
                            </span>
                        </div>

                        <div className="form-group">
                            <label className="label">Active Engine Driver</label>
                            <input
                                type="text"
                                className="input"
                                disabled
                                value={`${activeEngine.name} (${activeEngine.categoryLabel})`}
                                style={{ background: '#f8fafc', color: '#334155', cursor: 'default' }}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Dedicated driver binary used for network sockets, pooling, and query execution.
                            </span>
                        </div>
                    </div>
                </div>

                {/* Card 2: Network & Target Database (Context-Aware for each Database Engine) */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                            {isMongo ? 'MongoDB Cluster & Ingress Parameters' :
                             isMssql ? 'SQL Server Instance & Target Database' :
                             'Network & Target Database'}
                        </h2>

                        {isMongo && (
                            <div style={{ display: 'flex', gap: '6px' }}>
                                <button
                                    type="button"
                                    onClick={() => setMongoUriMode(false)}
                                    style={{
                                        padding: '4px 10px',
                                        fontSize: '11.5px',
                                        fontWeight: 600,
                                        borderRadius: '6px',
                                        border: !mongoUriMode ? '1px solid #2563eb' : '1px solid #e2e8f0',
                                        background: !mongoUriMode ? '#eff6ff' : '#ffffff',
                                        color: !mongoUriMode ? '#2563eb' : '#64748b',
                                        cursor: 'pointer'
                                    }}
                                >
                                    Standard Host/Port
                                </button>
                                <button
                                    type="button"
                                    onClick={() => setMongoUriMode(true)}
                                    style={{
                                        padding: '4px 10px',
                                        fontSize: '11.5px',
                                        fontWeight: 600,
                                        borderRadius: '6px',
                                        border: mongoUriMode ? '1px solid #2563eb' : '1px solid #e2e8f0',
                                        background: mongoUriMode ? '#eff6ff' : '#ffffff',
                                        color: mongoUriMode ? '#2563eb' : '#64748b',
                                        cursor: 'pointer'
                                    }}
                                >
                                    Connection String URI
                                </button>
                            </div>
                        )}
                    </div>

                    {/* SQL SERVER (MSSQL) CONFIGURATION */}
                    {isMssql && (
                        <>
                            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                                <div className="form-group">
                                    <label className="label">SQL Server Host / IP Address *</label>
                                    <input
                                        type="text"
                                        className="input"
                                        placeholder="e.g. localhost, mssql, or db.internal"
                                        required
                                        value={form.host}
                                        onChange={e => setForm(prev => ({ ...prev, host: e.target.value }))}
                                    />
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        Reachable SQL Server IP address or container hostname.
                                    </span>
                                </div>

                                <div className="form-group">
                                    <label className="label">Port Number *</label>
                                    <input
                                        type="number"
                                        className="input"
                                        placeholder="1433"
                                        required
                                        value={form.port}
                                        onChange={e => setForm(prev => ({ ...prev, port: parseInt(e.target.value, 10) || 0 }))}
                                    />
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        Standard SQL Server port is 1433.
                                    </span>
                                </div>
                            </div>

                            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                                <div className="form-group">
                                    <label className="label">Database Name *</label>
                                    <input
                                        type="text"
                                        className="input"
                                        placeholder="e.g. master or ProductionDB"
                                        required
                                        value={form.dbname}
                                        onChange={e => setForm(prev => ({ ...prev, dbname: e.target.value }))}
                                    />
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        Target catalog to query (e.g. <code>master</code>).
                                    </span>
                                </div>

                                <div className="form-group">
                                    <label className="label">Server Instance Name (Optional)</label>
                                    <input
                                        type="text"
                                        className="input"
                                        placeholder="e.g. SQLEXPRESS or leave empty"
                                        value={mssqlInstance}
                                        onChange={e => setMssqlInstance(e.target.value)}
                                    />
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        Named instance identifier if running multiple SQL Server engines.
                                    </span>
                                </div>
                            </div>

                            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                                <div className="form-group">
                                    <label className="label">SSL / Encryption Mode</label>
                                    <select
                                        className="select"
                                        value={form.sslmode}
                                        onChange={e => setForm(prev => ({ ...prev, sslmode: e.target.value }))}
                                    >
                                        <option value="disable">Disable (Standard local network)</option>
                                        <option value="require">Require (Encrypt SQL TDS stream)</option>
                                    </select>
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        Enforces Tabular Data Stream (TDS) encryption.
                                    </span>
                                </div>

                                <div className="form-group">
                                    <label className="label">Certificate Trust Policy</label>
                                    <div style={{ display: 'flex', alignItems: 'center', height: '38px', gap: '8px' }}>
                                        <input
                                            type="checkbox"
                                            id="trust-server-cert"
                                            checked={trustServerCert}
                                            onChange={e => setTrustServerCert(e.target.checked)}
                                            style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                                        />
                                        <label htmlFor="trust-server-cert" style={{ fontSize: '13px', color: '#1e293b', cursor: 'pointer', fontWeight: 500 }}>
                                            Trust Server Certificate (Recommended for self-signed certificates)
                                        </label>
                                    </div>
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        Prevents connection failures caused by internal self-signed TLS certificates.
                                    </span>
                                </div>
                            </div>
                        </>
                    )}

                    {/* MONGODB CONFIGURATION */}
                    {isMongo && (
                        mongoUriMode ? (
                            <div className="form-group" style={{ marginBottom: '16px' }}>
                                <label className="label">MongoDB Connection URI *</label>
                                <input
                                    type="text"
                                    className="input"
                                    placeholder="mongodb://username:password@cluster0.example.mongodb.net:27017/dbname?authSource=admin"
                                    value={mongoUri}
                                    onChange={e => handleMongoUriChange(e.target.value)}
                                />
                                <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                    Full MongoDB connection URI (supports standard <code>mongodb://</code> and Atlas SRV <code>mongodb+srv://</code>).
                                </span>
                            </div>
                        ) : (
                            <>
                                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                                    <div className="form-group">
                                        <label className="label">MongoDB Host / Cluster Node *</label>
                                        <input
                                            type="text"
                                            className="input"
                                            placeholder="localhost or mongodb.internal"
                                            required
                                            value={form.host}
                                            onChange={e => setForm(prev => ({ ...prev, host: e.target.value }))}
                                        />
                                        <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                            Primary MongoDB node hostname or cluster endpoint.
                                        </span>
                                    </div>

                                    <div className="form-group">
                                        <label className="label">Port Number *</label>
                                        <input
                                            type="number"
                                            className="input"
                                            placeholder="27017"
                                            required
                                            value={form.port}
                                            onChange={e => setForm(prev => ({ ...prev, port: parseInt(e.target.value, 10) || 0 }))}
                                        />
                                        <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                            Default MongoDB listening port is 27017.
                                        </span>
                                    </div>
                                </div>

                                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                                    <div className="form-group">
                                        <label className="label">Database Name *</label>
                                        <input
                                            type="text"
                                            className="input"
                                            placeholder="e.g. admin or production_db"
                                            required
                                            value={form.dbname}
                                            onChange={e => setForm(prev => ({ ...prev, dbname: e.target.value }))}
                                        />
                                        <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                            Target MongoDB database containing your document collections.
                                        </span>
                                    </div>

                                    <div className="form-group">
                                        <label className="label">Authentication Database (authSource)</label>
                                        <input
                                            type="text"
                                            className="input"
                                            placeholder="admin"
                                            value={form.auth_source}
                                            onChange={e => setForm(prev => ({ ...prev, auth_source: e.target.value }))}
                                        />
                                        <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                            MongoDB authentication database where credentials are registered (default: <code>admin</code>).
                                        </span>
                                    </div>
                                </div>
                            </>
                        )
                    )}

                    {/* POSTGRESQL, MYSQL & TRINO CONFIGURATION */}
                    {(isPostgres || isMysql || isTrino) && (
                        <>
                            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                                <div className="form-group">
                                    <label className="label">Host / IP Address *</label>
                                    <input
                                        type="text"
                                        className="input"
                                        placeholder={
                                            isPostgres ? 'e.g. postgres, 127.0.0.1, or db.internal' :
                                            isTrino ? 'e.g. trino, 127.0.0.1, or trino-coordinator' :
                                            'e.g. mysql, 127.0.0.1'
                                        }
                                        required
                                        value={form.host}
                                        onChange={e => setForm(prev => ({ ...prev, host: e.target.value }))}
                                    />
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        Reachable Docker service hostname or IP address.
                                    </span>
                                </div>

                                <div className="form-group">
                                    <label className="label">Port Number *</label>
                                    <input
                                        type="number"
                                        className="input"
                                        placeholder={String(activeEngine.port)}
                                        required
                                        value={form.port}
                                        onChange={e => setForm(prev => ({ ...prev, port: parseInt(e.target.value, 10) || 0 }))}
                                    />
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        Standard {activeEngine.name} listening port is {activeEngine.port}.
                                    </span>
                                </div>
                            </div>

                            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                                <div className="form-group">
                                    <label className="label">{isTrino ? 'Catalog / Schema *' : 'Database Name *'}</label>
                                    <input
                                        type="text"
                                        className="input"
                                        placeholder={isTrino ? 'e.g. tpch, tpch/sf1, or iceberg/analytics' : 'e.g. main_db or production'}
                                        required
                                        value={form.dbname}
                                        onChange={e => setForm(prev => ({ ...prev, dbname: e.target.value }))}
                                    />
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        {isTrino
                                            ? 'Trino catalog and optional schema (<catalog> or <catalog>/<schema>, defaults to schema default).'
                                            : 'Specific database or schema to inspect tables and execute queries on.'}
                                    </span>
                                </div>

                                <div className="form-group">
                                    <label className="label">SSL / Encryption Mode</label>
                                    <select
                                        className="select"
                                        value={form.sslmode}
                                        onChange={e => setForm(prev => ({ ...prev, sslmode: e.target.value }))}
                                    >
                                        <option value="disable">Disable (Standard local/docker network{isTrino ? ' - HTTP' : ''})</option>
                                        <option value="require">Require (Encrypted connection{isTrino ? ' - HTTPS' : ''})</option>
                                        {isPostgres && <option value="verify-ca">Verify-CA (Verify certificate authority)</option>}
                                        {isPostgres && <option value="verify-full">Verify-Full (Strict hostname verification)</option>}
                                    </select>
                                    <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                        TLS encryption enforcement for query transit.
                                    </span>
                                </div>
                            </div>
                        </>
                    )}
                </div>

                {/* Card 3: Authentication Credentials */}
                {(
                    <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                        <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                            {activeEngine.name} Authentication Credentials
                        </h2>

                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                            <div className="form-group">
                                <label className="label">Username *</label>
                                <input
                                    type="text"
                                    className="input"
                                    placeholder={
                                        isMongo ? 'e.g. admin or root' :
                                        isMssql ? 'sa' :
                                        isTrino ? 'trino' :
                                        isMysql ? 'root' : 'postgres'
                                    }
                                    required
                                    value={form.username}
                                    onChange={e => setForm(prev => ({ ...prev, username: e.target.value }))}
                                />
                                <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                    Database user with permissions to query target collections/tables.
                                </span>
                            </div>

                            <div className="form-group">
                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                    <label className="label">
                                        {`Password ${isEdit ? '(Leave blank to keep)' : isTrino ? '(Optional)' : '*'}`}
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
                                    placeholder={isEdit ? "•••••••• (Unchanged)" : isTrino ? "Leave blank if authentication is disabled" : "Database user password"}
                                    value={form.password}
                                    onChange={e => setForm(prev => ({ ...prev, password: e.target.value }))}
                                />
                                <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                    Encrypted securely in the gateway registry database.
                                </span>
                            </div>
                        </div>
                    </div>
                )}

                {/* Card 4: Pool & Safety Tuning */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: '0 0 16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        Connection Pool & Safety Controls
                    </h2>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
                        <div className="form-group">
                            <label className="label">Max Open Connections</label>
                            <input
                                type="number"
                                className="input"
                                min={1}
                                max={100}
                                value={form.max_open_conns}
                                onChange={e => setForm(prev => ({ ...prev, max_open_conns: parseInt(e.target.value, 10) || 10 }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Maximum concurrent database sockets allocated for this profile pool.
                            </span>
                        </div>

                        <div className="form-group">
                            <label className="label">Max Idle Connections</label>
                            <input
                                type="number"
                                className="input"
                                min={1}
                                max={50}
                                value={form.max_idle_conns}
                                onChange={e => setForm(prev => ({ ...prev, max_idle_conns: parseInt(e.target.value, 10) || 5 }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Idle connections maintained in cache to reduce handshake overhead.
                            </span>
                        </div>
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                        <div className="form-group">
                            <label className="label">Connection Max Lifetime (Seconds)</label>
                            <input
                                type="number"
                                className="input"
                                min={30}
                                max={86400}
                                value={form.conn_max_lifetime}
                                onChange={e => setForm(prev => ({ ...prev, conn_max_lifetime: parseInt(e.target.value, 10) || 300 }))}
                            />
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Maximum duration a single connection can be reused before cycling.
                            </span>
                        </div>

                        <div className="form-group">
                            <label className="label">Read-Only Safety Switch</label>
                            <div style={{ display: 'flex', alignItems: 'center', height: '38px', gap: '8px' }}>
                                <input
                                    type="checkbox"
                                    id="read-only-toggle"
                                    checked={form.is_read_only}
                                    onChange={e => setForm(prev => ({ ...prev, is_read_only: e.target.checked }))}
                                    style={{ width: '16px', height: '16px', cursor: 'pointer' }}
                                />
                                <label htmlFor="read-only-toggle" style={{ fontSize: '13px', color: '#1e293b', cursor: 'pointer', fontWeight: 500 }}>
                                    Enforce Read-Only Mode (Blocks write operations)
                                </label>
                            </div>
                            <span style={{ fontSize: '11.5px', color: '#64748b', marginTop: '4px', display: 'block' }}>
                                Protects your production data from accidental write or delete modifications.
                            </span>
                        </div>
                    </div>
                </div>

                {/* Card 5: Live Connectivity Diagnostics */}
                <div style={{ background: '#ffffff', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '10px' }}>
                        <h2 style={{ fontSize: '16px', fontWeight: 700, color: '#0f172a', margin: 0 }}>
                            Connectivity Diagnostics
                        </h2>
                        <button
                            type="button"
                            className="btn-header-refresh"
                            onClick={handleTestConnection}
                            disabled={isTesting}
                            style={{ display: 'flex', alignItems: 'center', gap: '6px' }}
                        >
                            <IconBolt width={14} height={14} />
                            <span>{isTesting ? 'Testing Connectivity...' : `Test ${activeEngine.name} Connection`}</span>
                        </button>
                    </div>

                    <p style={{ fontSize: '13px', color: '#64748b', margin: '0 0 16px 0' }}>
                        Perform a live handshake with the target {activeEngine.name} server to confirm network routing, credentials, and accessibility before saving.
                    </p>

                    {testResult && (
                        <div style={{
                            padding: '16px',
                            borderRadius: '8px',
                            background: testResult.success ? '#f0fdf4' : '#fef2f2',
                            border: `1px solid ${testResult.success ? '#bbf7d0' : '#fecaca'}`,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            gap: '12px'
                        }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                {testResult.success ? (
                                    <div style={{ width: '28px', height: '28px', borderRadius: '50%', background: '#dcfce7', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#16a34a' }}>
                                        <IconCheck width={16} height={16} />
                                    </div>
                                ) : (
                                    <div style={{ width: '28px', height: '28px', borderRadius: '50%', background: '#fee2e2', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#dc2626' }}>
                                        <IconAlertTriangle width={16} height={16} />
                                    </div>
                                )}
                                <div>
                                    <div style={{ fontSize: '13.5px', fontWeight: 600, color: testResult.success ? '#166534' : '#991b1b' }}>
                                        {testResult.success ? 'Connection Successful!' : 'Connection Failed'}
                                    </div>
                                    <div style={{ fontSize: '12px', color: testResult.success ? '#15803d' : '#b91c1c', marginTop: '2px' }}>
                                        {testResult.success
                                            ? `${activeEngine.name} responded smoothly with latency: ${testResult.latency_ms || 1} ms.`
                                            : testResult.error}
                                    </div>
                                </div>
                            </div>

                            {testResult.success && testResult.latency_ms !== undefined && (
                                <span style={{
                                    fontSize: '12px',
                                    fontWeight: 700,
                                    padding: '4px 10px',
                                    borderRadius: '6px',
                                    background: '#dcfce7',
                                    color: '#166534',
                                    border: '1px solid #86efac'
                                }}>
                                    {testResult.latency_ms} ms ping
                                </span>
                            )}
                        </div>
                    )}
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
                        onClick={() => navigate('/settings/connections')}
                        disabled={isSaving}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="btn-header-new-route"
                        disabled={isSaving}
                    >
                        {isSaving ? 'Saving...' : isEdit ? 'Update Connection' : `Save ${activeEngine.name} Connection`}
                    </button>
                </div>
            </form>

            {/* In-page Engine Switcher Modal */}
            <DatabaseEnginePickerModal
                isOpen={isPickerModalOpen}
                onClose={() => setIsPickerModalOpen(false)}
                onSelect={handleSelectEngineFromModal}
            />
        </div>
    );
}
