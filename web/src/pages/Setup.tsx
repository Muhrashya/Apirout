import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthProvider';
import { useBranding } from '../context/BrandingContext';
import { useToast } from '../components/Toast';
import './Login.css';

export default function Setup() {
    const { branding } = useBranding();
    const { completeSetup, setupRequired } = useAuth();
    const toast = useToast();
    const navigate = useNavigate();

    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [showConfirmPassword, setShowConfirmPassword] = useState(false);
    const [isLoading, setIsLoading] = useState(false);

    // If setup is already finished, immediately redirect to /login
    useEffect(() => {
        if (!setupRequired) {
            navigate('/login', { replace: true });
        }
    }, [setupRequired, navigate]);

    // Ensure light mode theme
    useEffect(() => {
        document.documentElement.setAttribute('data-theme', 'light');
        localStorage.setItem('theme', 'light');
    }, []);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        const cleanEmail = email.trim();
        if (!cleanEmail) {
            toast.error('Please enter an admin email address');
            return;
        }

        if (!password) {
            toast.error('Please enter a password');
            return;
        }

        if (password.length < 8) {
            toast.error('Password must be at least 8 characters long');
            return;
        }

        if (password !== confirmPassword) {
            toast.error('Passwords do not match');
            return;
        }

        setIsLoading(true);

        try {
            await completeSetup(cleanEmail, password, confirmPassword);
            toast.success('Super Admin initialized successfully! Welcome to Apirout.');
            navigate('/dashboard', { replace: true });
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to complete setup');
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="login-wrapper">
            <div className="login-card-container">
                <div className="login-card">
                    {/* Dynamic Brand Logo & Titles */}
                    <div className="login-titles">
                        <div className="login-logo-wrapper">
                            <img
                                src={branding.logo_url}
                                alt={branding.app_name}
                                className="login-logo-img"
                                onError={(e) => {
                                    (e.target as HTMLImageElement).src = '/logo.png';
                                }}
                            />
                        </div>
                        <h1 className="welcome-title">
                            Initial Setup
                        </h1>
                        <div className="welcome-tagline">
                            Create Super Admin
                        </div>
                    </div>

                    <div style={{
                        background: '#eff6ff',
                        border: '1px solid #bfdbfe',
                        borderRadius: '10px',
                        padding: '12px 14px',
                        marginBottom: '20px',
                        fontSize: '13px',
                        color: '#1e40af',
                        lineHeight: '1.45',
                        display: 'flex',
                        alignItems: 'flex-start',
                        gap: '8px'
                    }}>
                        <span style={{ fontSize: '15px' }}>ℹ️</span>
                        <span>Welcome! No administrator account exists yet. Create your root Super Admin credentials to secure and initialize this gateway instance.</span>
                    </div>

                    {/* Setup Form */}
                    <form onSubmit={handleSubmit} className="auth-form">
                        {/* Email Input */}
                        <div className="custom-input-group">
                            <input
                                id="setup-email"
                                type="email"
                                className="custom-input"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="admin@company.com"
                                autoComplete="email"
                                required
                            />
                        </div>

                        {/* Password Input with Show/Hide toggle */}
                        <div className="custom-input-group password-group">
                            <input
                                id="setup-password"
                                type={showPassword ? 'text' : 'password'}
                                className="custom-input"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                placeholder="Password (min 8 characters)"
                                autoComplete="new-password"
                                required
                            />
                            <button
                                type="button"
                                className="toggle-password-btn"
                                onClick={() => setShowPassword(!showPassword)}
                                tabIndex={-1}
                                aria-label={showPassword ? 'Hide password' : 'Show password'}
                            >
                                {showPassword ? 'Hide' : 'Show'}
                            </button>
                        </div>

                        {/* Confirm Password Input with Show/Hide toggle */}
                        <div className="custom-input-group password-group">
                            <input
                                id="setup-confirm-password"
                                type={showConfirmPassword ? 'text' : 'password'}
                                className="custom-input"
                                value={confirmPassword}
                                onChange={(e) => setConfirmPassword(e.target.value)}
                                placeholder="Confirm password"
                                autoComplete="new-password"
                                required
                            />
                            <button
                                type="button"
                                className="toggle-password-btn"
                                onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                                tabIndex={-1}
                                aria-label={showConfirmPassword ? 'Hide password' : 'Show password'}
                            >
                                {showConfirmPassword ? 'Hide' : 'Show'}
                            </button>
                        </div>

                        {/* Primary Submit Button */}
                        <button
                            type="submit"
                            className="primary-login-btn"
                            disabled={isLoading}
                        >
                            {isLoading ? (
                                <span className="btn-content-loading">
                                    <span className="spinner-icon"></span>
                                    <span>Initializing instance...</span>
                                </span>
                            ) : (
                                <span className="btn-content">
                                    <span>Initialize Super Admin</span>
                                    <span className="btn-arrow">→</span>
                                </span>
                            )}
                        </button>
                    </form>
                </div>
            </div>
        </div>
    );
}
