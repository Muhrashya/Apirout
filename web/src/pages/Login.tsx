import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthProvider';
import { useBranding } from '../context/BrandingContext';
import { useToast } from '../components/Toast';
import './Login.css';

export default function Login() {
    const { branding } = useBranding();
    const [email, setEmail] = useState('admin@apirout.io');
    const [password, setPassword] = useState('AdminApirout2026!');
    const [showPassword, setShowPassword] = useState(false);
    const [rememberMe, setRememberMe] = useState(true);
    const [isLoading, setIsLoading] = useState(false);

    const { login } = useAuth();
    const toast = useToast();
    const navigate = useNavigate();

    // Lock permanently to light mode
    useEffect(() => {
        document.documentElement.setAttribute('data-theme', 'light');
        localStorage.setItem('theme', 'light');
    }, []);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!email.trim() || !password) {
            toast.error('Please enter email and password');
            return;
        }

        setIsLoading(true);

        try {
            await login(email.trim(), password);
            toast.success('Login successful! Welcome back.');
            navigate('/dashboard', { replace: true });
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Invalid credentials');
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
                            {branding.app_name}
                        </h1>
                        <div className="welcome-tagline">
                            {branding.tagline}
                        </div>
                    </div>

                    {/* Login Form */}
                    <form onSubmit={handleSubmit} className="auth-form">
                        {/* Email Input */}
                        <div className="custom-input-group">
                            <input
                                id="email"
                                type="email"
                                className="custom-input"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="admin@apirout.io"
                                autoComplete="email"
                                required
                            />
                        </div>

                        {/* Password Input with Show/Hide toggle */}
                        <div className="custom-input-group password-group">
                            <input
                                id="password"
                                type={showPassword ? 'text' : 'password'}
                                className="custom-input"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                placeholder="••••••••"
                                autoComplete="current-password"
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

                        {/* Remember Me Row */}
                        <div className="form-options-row">
                            <label className="remember-me-label">
                                <input
                                    type="checkbox"
                                    checked={rememberMe}
                                    onChange={(e) => setRememberMe(e.target.checked)}
                                    className="custom-checkbox"
                                />
                                <span className="custom-checkbox-ui">
                                    <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                                        <polyline points="2.5 6 4.5 8.5 9.5 3.5" stroke="#ffffff" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                                    </svg>
                                </span>
                                <span className="label-text">Remember me</span>
                            </label>
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
                                    <span>Signing in...</span>
                                </span>
                            ) : (
                                <span className="btn-content">
                                    <span>Log in</span>
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

