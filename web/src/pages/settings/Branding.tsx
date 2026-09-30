import React, { useState, useEffect, useRef } from 'react';
import { useBranding } from '../../context/BrandingContext';
import { useToast } from '../../components/Toast';
import {
    IconUpload,
    IconRefresh,
    IconCheck,
    IconEye
} from '../../components/Icons';

export default function BrandingPage() {
    const { branding, updateBranding, resetBranding } = useBranding();
    const toast = useToast();
    const fileInputRef = useRef<HTMLInputElement>(null);

    const [form, setForm] = useState({
        app_name: branding.app_name,
        tagline: branding.tagline,
        logo_url: branding.logo_url,
        favicon_url: branding.favicon_url,
        browser_title: branding.browser_title,
    });

    const [useLogoAsFavicon, setUseLogoAsFavicon] = useState(true);
    const [isSaving, setIsSaving] = useState(false);

    useEffect(() => {
        setForm({
            app_name: branding.app_name,
            tagline: branding.tagline,
            logo_url: branding.logo_url,
            favicon_url: branding.favicon_url,
            browser_title: branding.browser_title,
        });
        setUseLogoAsFavicon(branding.favicon_url === branding.logo_url || branding.favicon_url === '/favicon.png' || !branding.favicon_url);
    }, [branding]);

    const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        if (file.size > 2 * 1024 * 1024) {
            toast.error('Image size must be less than 2MB');
            return;
        }

        const reader = new FileReader();
        reader.onload = () => {
            const dataUrl = reader.result as string;
            setForm(prev => ({
                ...prev,
                logo_url: dataUrl,
                favicon_url: useLogoAsFavicon ? dataUrl : prev.favicon_url,
            }));
            toast.success('Logo loaded from file. Click "Save Changes" to apply.');
        };
        reader.readAsDataURL(file);
    };

    const handleResetLogoToDefault = () => {
        setForm(prev => ({
            ...prev,
            logo_url: '/logo.png',
            favicon_url: useLogoAsFavicon ? '/favicon.png' : prev.favicon_url,
        }));
        toast.info('Logo set to default Apirout logo.');
    };

    const handleSave = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!form.app_name.trim()) {
            toast.error('Application name cannot be empty');
            return;
        }

        setIsSaving(true);
        try {
            const payload = {
                app_name: form.app_name.trim(),
                tagline: form.tagline.trim(),
                logo_url: form.logo_url.trim() || '/logo.png',
                favicon_url: useLogoAsFavicon ? (form.logo_url.trim() || '/favicon.png') : (form.favicon_url.trim() || '/favicon.png'),
                browser_title: form.browser_title.trim() || `${form.app_name.trim()} Console`,
            };

            await updateBranding(payload);
            toast.success('Branding and white-label settings updated successfully!');
        } catch (err: any) {
            toast.error(err.message || 'Failed to save branding settings');
        } finally {
            setIsSaving(false);
        }
    };

    const handleResetAll = async () => {
        if (!window.confirm('Reset all branding settings (logo, app name, tagline, favicon) back to factory defaults?')) {
            return;
        }
        setIsSaving(true);
        try {
            await resetBranding();
            toast.success('Branding reset to default factory settings');
        } catch (err: any) {
            toast.error(err.message || 'Failed to reset branding');
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <div className="overview-page-wrapper">
            {/* Breadcrumb */}
            <div className="overview-breadcrumb">
                <span>Settings</span>
                <span className="breadcrumb-sep">/</span>
                <span className="breadcrumb-curr">Customization &amp; Branding</span>
            </div>

            {/* Page Header */}
            <div className="overview-header" style={{ marginBottom: '24px' }}>
                <div className="overview-title-col">
                    <h1 className="overview-title">Customization &amp; Branding</h1>
                    <p className="overview-subtitle">
                        White-label and personalize your API Gateway. Change the logo, platform name, tagline, favicon, and browser tab naming.
                    </p>
                </div>

                <div className="overview-header-actions">
                    <button
                        type="button"
                        className="btn-header-refresh"
                        onClick={handleResetAll}
                        disabled={isSaving}
                        title="Reset all settings to default"
                    >
                        <IconRefresh width={14} height={14} />
                        <span>Reset Defaults</span>
                    </button>
                    <button
                        type="button"
                        className="btn-header-new-route"
                        onClick={handleSave}
                        disabled={isSaving}
                    >
                        <IconCheck width={15} height={15} />
                        <span>{isSaving ? 'Saving...' : 'Save Changes'}</span>
                    </button>
                </div>
            </div>

            {/* 2-Column Responsive Layout: Form on Left, Live Preview on Right */}
            <div style={{
                display: 'grid',
                gridTemplateColumns: 'minmax(0, 1.4fr) minmax(0, 1.1fr)',
                gap: '24px',
                alignItems: 'start'
            }}>
                {/* LEFT: FORM CARD */}
                <div style={{
                    backgroundColor: '#ffffff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '12px',
                    padding: '24px',
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                }}>
                    <form onSubmit={handleSave} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
                        {/* Application Name */}
                        <div>
                            <label style={{ display: 'block', fontSize: '13.5px', fontWeight: 600, color: '#1e293b', marginBottom: '6px' }}>
                                Application Name *
                            </label>
                            <input
                                type="text"
                                placeholder="e.g. Apirout, Acme Gateway, Nexus API"
                                value={form.app_name}
                                onChange={e => {
                                    const val = e.target.value;
                                    setForm(prev => ({
                                        ...prev,
                                        app_name: val,
                                        browser_title: prev.browser_title === `${prev.app_name} Console` ? `${val} Console` : prev.browser_title
                                    }));
                                }}
                                required
                                style={{
                                    width: '100%',
                                    padding: '9px 14px',
                                    fontSize: '14px',
                                    border: '1px solid #cbd5e1',
                                    borderRadius: '8px',
                                    outline: 'none',
                                    boxSizing: 'border-box'
                                }}
                            />
                            <p style={{ fontSize: '12px', color: '#64748b', margin: '4px 0 0 0' }}>
                                Displayed in the top navigation bar, login screen, and page headers.
                            </p>
                        </div>

                        {/* Tagline / Subtitle */}
                        <div>
                            <label style={{ display: 'block', fontSize: '13.5px', fontWeight: 600, color: '#1e293b', marginBottom: '6px' }}>
                                Tagline / Subtitle
                            </label>
                            <input
                                type="text"
                                placeholder="e.g. API GATEWAY, ENTERPRISE ENGINE"
                                value={form.tagline}
                                onChange={e => setForm(prev => ({ ...prev, tagline: e.target.value }))}
                                style={{
                                    width: '100%',
                                    padding: '9px 14px',
                                    fontSize: '14px',
                                    border: '1px solid #cbd5e1',
                                    borderRadius: '8px',
                                    outline: 'none',
                                    boxSizing: 'border-box'
                                }}
                            />
                            <p style={{ fontSize: '12px', color: '#64748b', margin: '4px 0 0 0' }}>
                                Small label placed below the application name in uppercase.
                            </p>
                        </div>

                        {/* Logo Upload & Configuration */}
                        <div>
                            <label style={{ display: 'block', fontSize: '13.5px', fontWeight: 600, color: '#1e293b', marginBottom: '6px' }}>
                                Logo Image
                            </label>
                            <div style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: '16px',
                                padding: '16px',
                                border: '1px solid #e2e8f0',
                                borderRadius: '10px',
                                backgroundColor: '#f8fafc',
                                marginBottom: '10px'
                            }}>
                                {/* Logo Thumbnail Box */}
                                <div style={{
                                    width: '64px',
                                    height: '64px',
                                    borderRadius: '10px',
                                    backgroundColor: '#ffffff',
                                    border: '1px solid #cbd5e1',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    overflow: 'hidden',
                                    flexShrink: 0,
                                    padding: '6px',
                                    boxSizing: 'border-box'
                                }}>
                                    <img
                                        src={form.logo_url}
                                        alt="Logo preview"
                                        style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain' }}
                                        onError={(e) => {
                                            (e.target as HTMLImageElement).src = '/logo.png';
                                        }}
                                    />
                                </div>

                                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', flex: 1 }}>
                                    <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
                                        <input
                                            ref={fileInputRef}
                                            type="file"
                                            accept="image/png, image/jpeg, image/svg+xml, image/webp"
                                            onChange={handleFileUpload}
                                            style={{ display: 'none' }}
                                        />
                                        <button
                                            type="button"
                                            onClick={() => fileInputRef.current?.click()}
                                            style={{
                                                display: 'inline-flex',
                                                alignItems: 'center',
                                                gap: '6px',
                                                padding: '7px 12px',
                                                fontSize: '13px',
                                                fontWeight: 600,
                                                color: '#2563eb',
                                                backgroundColor: '#eff6ff',
                                                border: '1px solid #bfdbfe',
                                                borderRadius: '6px',
                                                cursor: 'pointer'
                                            }}
                                        >
                                            <IconUpload width={14} height={14} />
                                            <span>Upload New Logo</span>
                                        </button>

                                        <button
                                            type="button"
                                            onClick={handleResetLogoToDefault}
                                            style={{
                                                padding: '7px 12px',
                                                fontSize: '13px',
                                                fontWeight: 500,
                                                color: '#475569',
                                                backgroundColor: '#ffffff',
                                                border: '1px solid #cbd5e1',
                                                borderRadius: '6px',
                                                cursor: 'pointer'
                                            }}
                                        >
                                            Use Default Logo
                                        </button>
                                    </div>
                                    <span style={{ fontSize: '12px', color: '#64748b' }}>
                                        Recommended: PNG with transparent background, square or horizontal aspect ratio (max 2MB).
                                    </span>
                                </div>
                            </div>

                            {/* Direct URL input option */}
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <span style={{ fontSize: '12px', color: '#64748b', whiteSpace: 'nowrap' }}>Or Image URL:</span>
                                <input
                                    type="text"
                                    placeholder="/logo.png or https://example.com/logo.png"
                                    value={form.logo_url.startsWith('data:') ? '(Uploaded Base64 Image)' : form.logo_url}
                                    onChange={e => {
                                        const val = e.target.value;
                                        setForm(prev => ({
                                            ...prev,
                                            logo_url: val,
                                            favicon_url: useLogoAsFavicon ? val : prev.favicon_url,
                                        }));
                                    }}
                                    disabled={form.logo_url.startsWith('data:')}
                                    style={{
                                        flex: 1,
                                        padding: '6px 10px',
                                        fontSize: '13px',
                                        border: '1px solid #cbd5e1',
                                        borderRadius: '6px',
                                        outline: 'none',
                                        backgroundColor: form.logo_url.startsWith('data:') ? '#f1f5f9' : '#ffffff'
                                    }}
                                />
                                {form.logo_url.startsWith('data:') && (
                                    <button
                                        type="button"
                                        onClick={() => setForm(prev => ({ ...prev, logo_url: '/logo.png' }))}
                                        style={{ fontSize: '12px', color: '#dc2626', background: 'none', border: 'none', cursor: 'pointer' }}
                                    >
                                        Clear Upload
                                    </button>
                                )}
                            </div>
                        </div>

                        {/* Favicon Setting */}
                        <div>
                            <label style={{ display: 'block', fontSize: '13.5px', fontWeight: 600, color: '#1e293b', marginBottom: '6px' }}>
                                Browser Favicon
                            </label>
                            <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer', marginBottom: '8px' }}>
                                <input
                                    type="checkbox"
                                    checked={useLogoAsFavicon}
                                    onChange={e => {
                                        const checked = e.target.checked;
                                        setUseLogoAsFavicon(checked);
                                        if (checked) {
                                            setForm(prev => ({ ...prev, favicon_url: prev.logo_url }));
                                        }
                                    }}
                                    style={{ width: '16px', height: '16px', accentColor: '#2563eb' }}
                                />
                                <span style={{ fontSize: '13px', color: '#334155' }}>
                                    Automatically synchronize favicon with the active logo
                                </span>
                            </label>

                            {!useLogoAsFavicon && (
                                <input
                                    type="text"
                                    placeholder="Favicon URL (e.g. /favicon.png)"
                                    value={form.favicon_url}
                                    onChange={e => setForm(prev => ({ ...prev, favicon_url: e.target.value }))}
                                    style={{
                                        width: '100%',
                                        padding: '8px 12px',
                                        fontSize: '13.5px',
                                        border: '1px solid #cbd5e1',
                                        borderRadius: '6px',
                                        outline: 'none',
                                        boxSizing: 'border-box'
                                    }}
                                />
                            )}
                        </div>

                        {/* Browser Tab Title */}
                        <div>
                            <label style={{ display: 'block', fontSize: '13.5px', fontWeight: 600, color: '#1e293b', marginBottom: '6px' }}>
                                Browser Tab Title *
                            </label>
                            <input
                                type="text"
                                placeholder="e.g. Apirout Console, Acme API Portal"
                                value={form.browser_title}
                                onChange={e => setForm(prev => ({ ...prev, browser_title: e.target.value }))}
                                required
                                style={{
                                    width: '100%',
                                    padding: '9px 14px',
                                    fontSize: '14px',
                                    border: '1px solid #cbd5e1',
                                    borderRadius: '8px',
                                    outline: 'none',
                                    boxSizing: 'border-box'
                                }}
                            />
                            <p style={{ fontSize: '12px', color: '#64748b', margin: '4px 0 0 0' }}>
                                The text that appears on browser tabs and bookmarks.
                            </p>
                        </div>

                        {/* Submit Button */}
                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '10px' }}>
                            <button
                                type="submit"
                                disabled={isSaving}
                                className="btn-header-new-route"
                                style={{ padding: '10px 20px', fontSize: '14px' }}
                            >
                                <IconCheck width={16} height={16} />
                                <span>{isSaving ? 'Saving Changes...' : 'Save & Apply Branding'}</span>
                            </button>
                        </div>
                    </form>
                </div>

                {/* RIGHT: LIVE PREVIEW PANEL */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
                    {/* Panel Header */}
                    <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
                        fontSize: '14px',
                        fontWeight: 700,
                        color: '#0f172a'
                    }}>
                        <IconEye width={18} height={18} style={{ color: '#2563eb' }} />
                        <span>Live Preview</span>
                    </div>

                    {/* Preview 1: Browser Tab Mockup */}
                    <div style={{
                        backgroundColor: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '10px',
                        overflow: 'hidden',
                        boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                    }}>
                        <div style={{
                            padding: '8px 12px',
                            background: '#e2e8f0',
                            borderBottom: '1px solid #cbd5e1',
                            fontSize: '12px',
                            fontWeight: 600,
                            color: '#475569'
                        }}>
                            Browser Tab Appearance
                        </div>
                        <div style={{ padding: '12px 16px', background: '#f8fafc' }}>
                            <div style={{
                                display: 'inline-flex',
                                alignItems: 'center',
                                gap: '8px',
                                padding: '6px 14px',
                                background: '#ffffff',
                                border: '1px solid #cbd5e1',
                                borderRadius: '8px 8px 0 0',
                                boxShadow: '0 -1px 3px rgba(0,0,0,0.03)'
                            }}>
                                <img
                                    src={useLogoAsFavicon ? form.logo_url : form.favicon_url}
                                    alt="favicon"
                                    style={{ width: '16px', height: '16px', objectFit: 'contain' }}
                                    onError={(e) => { (e.target as HTMLImageElement).src = '/logo.png'; }}
                                />
                                <span style={{ fontSize: '12.5px', fontWeight: 600, color: '#1e293b' }}>
                                    {form.browser_title || `${form.app_name} Console`}
                                </span>
                            </div>
                        </div>
                    </div>

                    {/* Preview 2: Navbar Brand Mockup */}
                    <div style={{
                        backgroundColor: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '10px',
                        overflow: 'hidden',
                        boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                    }}>
                        <div style={{
                            padding: '8px 12px',
                            background: '#e2e8f0',
                            borderBottom: '1px solid #cbd5e1',
                            fontSize: '12px',
                            fontWeight: 600,
                            color: '#475569'
                        }}>
                            Navbar Header Preview
                        </div>
                        <div style={{
                            padding: '12px 20px',
                            background: '#ffffff',
                            borderBottom: '1px solid #e2e8f0',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between'
                        }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                                <div style={{
                                    width: '36px',
                                    height: '36px',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    overflow: 'hidden'
                                }}>
                                    <img
                                        src={form.logo_url}
                                        alt="brand preview"
                                        style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain' }}
                                        onError={(e) => { (e.target as HTMLImageElement).src = '/logo.png'; }}
                                    />
                                </div>
                                <div>
                                    <div style={{ fontSize: '15px', fontWeight: 800, color: '#0f172a', lineHeight: 1.1 }}>
                                        {form.app_name || 'Apirout'}
                                    </div>
                                    <div style={{ fontSize: '9px', fontWeight: 700, color: '#64748b', letterSpacing: '0.8px', marginTop: '2px' }}>
                                        {form.tagline || 'API GATEWAY'}
                                    </div>
                                </div>
                            </div>

                            <div style={{
                                width: '32px',
                                height: '32px',
                                borderRadius: '50%',
                                background: '#eff6ff',
                                color: '#2563eb',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                fontSize: '12px',
                                fontWeight: 700
                            }}>
                                AD
                            </div>
                        </div>
                    </div>

                    {/* Preview 3: Login Screen Mockup */}
                    <div style={{
                        backgroundColor: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '10px',
                        overflow: 'hidden',
                        boxShadow: '0 1px 3px rgba(0, 0, 0, 0.05)'
                    }}>
                        <div style={{
                            padding: '8px 12px',
                            background: '#e2e8f0',
                            borderBottom: '1px solid #cbd5e1',
                            fontSize: '12px',
                            fontWeight: 600,
                            color: '#475569'
                        }}>
                            Login Screen Preview
                        </div>
                        <div style={{
                            padding: '30px 24px',
                            background: '#f8fafc',
                            textAlign: 'center'
                        }}>
                            <img
                                src={form.logo_url}
                                alt="Login brand preview"
                                style={{
                                    width: '72px',
                                    height: '72px',
                                    objectFit: 'contain',
                                    margin: '0 auto 10px auto',
                                    display: 'block',
                                    filter: 'drop-shadow(0 4px 12px rgba(37, 99, 235, 0.15))'
                                }}
                                onError={(e) => { (e.target as HTMLImageElement).src = '/logo.png'; }}
                            />
                            <div style={{ fontSize: '22px', fontWeight: 800, color: '#0f172a', margin: '0 0 4px 0' }}>
                                {form.app_name || 'Apirout'}
                            </div>
                            <div style={{ fontSize: '11px', fontWeight: 700, color: '#2563eb', letterSpacing: '1.2px', textTransform: 'uppercase', marginBottom: '16px' }}>
                                {form.tagline || 'API GATEWAY'}
                            </div>

                            {/* Mock Input fields */}
                            <div style={{ maxWidth: '240px', margin: '0 auto', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                <div style={{ height: '30px', background: '#ffffff', border: '1px solid #cbd5e1', borderRadius: '6px' }} />
                                <div style={{ height: '30px', background: '#ffffff', border: '1px solid #cbd5e1', borderRadius: '6px' }} />
                                <div style={{ height: '32px', background: '#2563eb', borderRadius: '6px' }} />
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
