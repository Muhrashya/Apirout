import { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
    children: ReactNode;
    fallback?: ReactNode;
}

interface State {
    hasError: boolean;
    error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
    public state: State = {
        hasError: false,
        error: null,
    };

    public static getDerivedStateFromError(error: Error): State {
        return { hasError: true, error };
    }

    public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        console.error('Uncaught error in component tree:', error, errorInfo);
    }

    public render() {
        if (this.state.hasError) {
            if (this.fallback) {
                return this.fallback;
            }

            return (
                <div style={{
                    padding: '40px 24px',
                    maxWidth: '800px',
                    margin: '40px auto',
                    backgroundColor: '#ffffff',
                    borderRadius: '12px',
                    border: '1px solid #fee2e2',
                    boxShadow: '0 4px 12px rgba(239, 68, 68, 0.08)',
                    textAlign: 'center',
                    fontFamily: 'Inter, sans-serif'
                }}>
                    <div style={{
                        width: '56px',
                        height: '56px',
                        borderRadius: '50%',
                        backgroundColor: '#fef2f2',
                        color: '#ef4444',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '28px',
                        margin: '0 auto 16px auto',
                    }}>
                        ⚠️
                    </div>
                    <h2 style={{ fontSize: '20px', fontWeight: 700, color: '#0f172a', marginBottom: '8px' }}>
                        Something went wrong
                    </h2>
                    <p style={{ fontSize: '14px', color: '#64748b', marginBottom: '20px', maxWidth: '500px', margin: '0 auto 20px auto' }}>
                        {this.state.error?.message || 'An unexpected rendering error occurred. Please try reloading this section.'}
                    </p>
                    <div style={{ display: 'flex', gap: '12px', justifyContent: 'center' }}>
                        <button
                            onClick={() => this.setState({ hasError: false, error: null })}
                            style={{
                                padding: '8px 18px',
                                background: '#2563eb',
                                color: '#ffffff',
                                border: 'none',
                                borderRadius: '8px',
                                fontSize: '14px',
                                fontWeight: 500,
                                cursor: 'pointer',
                            }}
                        >
                            Try Again
                        </button>
                        <button
                            onClick={() => window.location.reload()}
                            style={{
                                padding: '8px 18px',
                                background: '#f8fafc',
                                color: '#334155',
                                border: '1px solid #e2e8f0',
                                borderRadius: '8px',
                                fontSize: '14px',
                                fontWeight: 500,
                                cursor: 'pointer',
                            }}
                        >
                            Reload Page
                        </button>
                    </div>
                </div>
            );
        }

        return this.props.children;
    }

    private get fallback(): ReactNode | undefined {
        return this.props.fallback;
    }
}
export default ErrorBoundary;
