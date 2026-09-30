import React, { useState } from 'react';
import { IconSearch } from './Icons';

import logoMongodb from '../assets/logos/logo-mongodb.png';
import logoMysql from '../assets/logos/logo-mysql.png';
import logoPostgresql from '../assets/logos/logo-postgresql.png';
import logoSqlserver from '../assets/logos/logo-sqlserver.png';

const IconClose = () => (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <line x1="18" y1="6" x2="6" y2="18"></line>
        <line x1="6" y1="6" x2="18" y2="18"></line>
    </svg>
);

export interface DatabaseEngineOption {
    id: string;
    name: string;
    category: 'sql' | 'nosql';
    categoryLabel: string;
    badge?: string;
    badgeColor?: { bg: string; text: string; border: string };
    port: number;
    description: string;
    icon: React.ReactNode;
}

export const DATABASE_ENGINES: DatabaseEngineOption[] = [
    {
        id: 'postgres',
        name: 'PostgreSQL',
        category: 'sql',
        categoryLabel: 'Relational SQL',
        port: 5432,
        description: 'Advanced open-source relational database with full Instant REST API & SQL query builder.',
        icon: <img src={logoPostgresql} alt="PostgreSQL" style={{ width: '38px', height: '38px', objectFit: 'contain' }} />
    },
    {
        id: 'mysql',
        name: 'MySQL',
        category: 'sql',
        categoryLabel: 'Relational SQL',
        port: 3306,
        description: 'World-renowned high performance relational database powering scalable web applications.',
        icon: <img src={logoMysql} alt="MySQL" style={{ width: '38px', height: '38px', objectFit: 'contain' }} />
    },
    {
        id: 'mongodb',
        name: 'MongoDB',
        category: 'nosql',
        categoryLabel: 'Document NoSQL',
        port: 27017,
        description: 'Flexible JSON-like document store. Expose collections automatically as Instant REST APIs.',
        icon: <img src={logoMongodb} alt="MongoDB" style={{ width: '38px', height: '38px', objectFit: 'contain' }} />
    },
    {
        id: 'mssql',
        name: 'SQL Server',
        category: 'sql',
        categoryLabel: 'Enterprise SQL',
        port: 1433,
        description: 'Microsoft enterprise database engine with robust T-SQL, transaction control, and pooling.',
        icon: <img src={logoSqlserver} alt="SQL Server" style={{ width: '38px', height: '38px', objectFit: 'contain' }} />
    }
];

interface DatabaseEnginePickerModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSelect: (engineId: string) => void;
}

export const DatabaseEnginePickerModal: React.FC<DatabaseEnginePickerModalProps> = ({
    isOpen,
    onClose,
    onSelect
}) => {
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedCategory, setSelectedCategory] = useState<'all' | 'sql' | 'nosql'>('all');

    if (!isOpen) return null;

    const filteredEngines = DATABASE_ENGINES.filter(engine => {
        const matchesQuery =
            engine.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            engine.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
            engine.categoryLabel.toLowerCase().includes(searchQuery.toLowerCase()) ||
            engine.id.toLowerCase().includes(searchQuery.toLowerCase());
        const matchesCategory = selectedCategory === 'all' || engine.category === selectedCategory;
        return matchesQuery && matchesCategory;
    });

    return (
        <div style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(15, 23, 42, 0.65)',
            backdropFilter: 'blur(4px)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000,
            padding: '20px'
        }}>
            <div style={{
                backgroundColor: '#ffffff',
                borderRadius: '16px',
                width: '100%',
                maxWidth: '820px',
                maxHeight: '90vh',
                display: 'flex',
                flexDirection: 'column',
                boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.2), 0 10px 10px -5px rgba(0, 0, 0, 0.1)',
                border: '1px solid #e2e8f0',
                overflow: 'hidden'
            }}>
                {/* Header */}
                <div style={{
                    padding: '24px 28px 18px',
                    borderBottom: '1px solid #f1f5f9',
                    display: 'flex',
                    alignItems: 'flex-start',
                    justifyContent: 'space-between'
                }}>
                    <div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <span style={{
                                display: 'inline-flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                width: '28px',
                                height: '28px',
                                borderRadius: '8px',
                                background: '#eff6ff',
                                color: '#2563eb'
                            }}>
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                                    <ellipse cx="12" cy="5" rx="9" ry="3"></ellipse>
                                    <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path>
                                    <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>
                                </svg>
                            </span>
                            <h2 style={{ margin: 0, fontSize: '19px', fontWeight: 700, color: '#0f172a' }}>
                                Select Database Driver &amp; Engine
                            </h2>
                        </div>
                        <p style={{ margin: '6px 0 0 36px', fontSize: '13px', color: '#64748b' }}>
                            Choose a database source to configure connection pooling, automated Instant REST APIs, and query proxying.
                        </p>
                    </div>

                    <button
                        onClick={onClose}
                        style={{
                            background: '#f8fafc',
                            border: '1px solid #e2e8f0',
                            borderRadius: '8px',
                            cursor: 'pointer',
                            padding: '6px',
                            color: '#64748b',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            transition: 'all 0.15s'
                        }}
                        title="Close modal"
                    >
                        <IconClose />
                    </button>
                </div>

                {/* Filter & Search Bar */}
                <div style={{
                    padding: '16px 28px',
                    background: '#f8fafc',
                    borderBottom: '1px solid #e2e8f0',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    flexWrap: 'wrap',
                    gap: '12px'
                }}>
                    {/* Category Tabs */}
                    <div style={{ display: 'flex', gap: '6px' }}>
                        {[
                            { id: 'all', label: 'All Databases' },
                            { id: 'sql', label: 'Relational (SQL)' },
                            { id: 'nosql', label: 'Document (NoSQL)' }
                        ].map(cat => (
                            <button
                                key={cat.id}
                                onClick={() => setSelectedCategory(cat.id as any)}
                                style={{
                                    padding: '6px 12px',
                                    borderRadius: '6px',
                                    fontSize: '12.5px',
                                    fontWeight: 600,
                                    border: selectedCategory === cat.id ? '1px solid #2563eb' : '1px solid #e2e8f0',
                                    background: selectedCategory === cat.id ? '#2563eb' : '#ffffff',
                                    color: selectedCategory === cat.id ? '#ffffff' : '#475569',
                                    cursor: 'pointer',
                                    transition: 'all 0.15s ease'
                                }}
                            >
                                {cat.label}
                            </button>
                        ))}
                    </div>

                    {/* Search Input */}
                    <div style={{ position: 'relative', minWidth: '240px', flex: '1 1 240px', maxWidth: '320px' }}>
                        <span style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', color: '#94a3b8', display: 'flex' }}>
                            <IconSearch width={14} height={14} />
                        </span>
                        <input
                            type="text"
                            placeholder="Filter engines (e.g. postgres, mongo)..."
                            value={searchQuery}
                            onChange={e => setSearchQuery(e.target.value)}
                            style={{
                                width: '100%',
                                padding: '7px 12px 7px 30px',
                                borderRadius: '8px',
                                border: '1px solid #cbd5e1',
                                fontSize: '13px',
                                outline: 'none',
                                background: '#ffffff'
                            }}
                        />
                    </div>
                </div>

                {/* Database Engine Cards Grid */}
                <div style={{
                    padding: '24px 28px',
                    overflowY: 'auto',
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))',
                    gap: '16px',
                    maxHeight: '520px'
                }}>
                    {filteredEngines.map(engine => (
                        <div
                            key={engine.id}
                            onClick={() => onSelect(engine.id)}
                            style={{
                                border: '1.5px solid #e2e8f0',
                                borderRadius: '12px',
                                padding: '18px',
                                background: '#ffffff',
                                cursor: 'pointer',
                                display: 'flex',
                                flexDirection: 'column',
                                justifyContent: 'space-between',
                                transition: 'all 0.18s cubic-bezier(0.16, 1, 0.3, 1)',
                                position: 'relative'
                            }}
                            onMouseEnter={e => {
                                e.currentTarget.style.borderColor = '#2563eb';
                                e.currentTarget.style.boxShadow = '0 8px 16px -4px rgba(37, 99, 235, 0.12)';
                                e.currentTarget.style.transform = 'translateY(-2px)';
                            }}
                            onMouseLeave={e => {
                                e.currentTarget.style.borderColor = '#e2e8f0';
                                e.currentTarget.style.boxShadow = 'none';
                                e.currentTarget.style.transform = 'translateY(0)';
                            }}
                        >
                            <div>
                                <div style={{ display: 'flex', alignItems: 'center', marginBottom: '14px' }}>
                                    <div style={{
                                        width: '46px',
                                        height: '46px',
                                        borderRadius: '10px',
                                        background: '#f8fafc',
                                        border: '1px solid #f1f5f9',
                                        display: 'flex',
                                        alignItems: 'center',
                                        justifyContent: 'center',
                                        flexShrink: 0,
                                        padding: '4px'
                                    }}>
                                        {engine.icon}
                                    </div>
                                </div>

                                <div style={{ fontSize: '15px', fontWeight: 700, color: '#0f172a', marginBottom: '4px' }}>
                                    {engine.name}
                                </div>

                                <div style={{ fontSize: '11.5px', color: '#64748b', fontWeight: 600, marginBottom: '8px' }}>
                                    {engine.categoryLabel} {engine.port > 0 ? `• Port ${engine.port}` : ''}
                                </div>

                                <p style={{ fontSize: '12px', color: '#64748b', margin: 0, lineHeight: 1.45 }}>
                                    {engine.description}
                                </p>
                            </div>

                            <div style={{
                                marginTop: '16px',
                                paddingTop: '12px',
                                borderTop: '1px solid #f1f5f9',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'space-between',
                                fontSize: '12.5px',
                                fontWeight: 600,
                                color: '#2563eb'
                            }}>
                                <span>Configure Driver</span>
                                <span>&rarr;</span>
                            </div>
                        </div>
                    ))}

                    {filteredEngines.length === 0 && (
                        <div style={{ gridColumn: '1 / -1', padding: '48px 20px', textAlign: 'center', color: '#64748b' }}>
                            <p style={{ margin: 0, fontSize: '14px', fontWeight: 500 }}>
                                No database drivers found matching "{searchQuery}".
                            </p>
                        </div>
                    )}
                </div>

                {/* Footer Info */}
                <div style={{
                    padding: '14px 28px',
                    background: '#f8fafc',
                    borderTop: '1px solid #e2e8f0',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    fontSize: '12px',
                    color: '#64748b'
                }}>
                    <span>Tip: DBeaver-style database catalog with official pre-configured driver defaults.</span>
                    <button
                        onClick={onClose}
                        style={{
                            background: '#ffffff',
                            border: '1px solid #cbd5e1',
                            borderRadius: '6px',
                            padding: '6px 14px',
                            fontSize: '12.5px',
                            fontWeight: 600,
                            color: '#475569',
                            cursor: 'pointer'
                        }}
                    >
                        Cancel
                    </button>
                </div>
            </div>
        </div>
    );
};
