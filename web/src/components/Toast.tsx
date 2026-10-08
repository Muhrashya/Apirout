import { createContext, useContext, useState, useCallback, useRef, ReactNode } from 'react';
import { createPortal } from 'react-dom';
import './Toast.css';

type ToastType = 'success' | 'error' | 'warning' | 'info';

interface Toast {
    id: string;
    type: ToastType;
    message: string;
}

interface ToastContextType {
    showToast: (type: ToastType, message: string) => void;
    success: (message: string) => void;
    error: (message: string) => void;
    warning: (message: string) => void;
    info: (message: string) => void;
}

const ToastContext = createContext<ToastContextType | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
    const [toasts, setToasts] = useState<Toast[]>([]);
    const recentDispatches = useRef<Map<string, number>>(new Map());

    const showToast = useCallback((type: ToastType, message: string) => {
        const key = `${type}:${message}`;
        const now = Date.now();
        const lastTime = recentDispatches.current.get(key) || 0;

        // 1,000ms window debounce: suppress identical toast pills dispatched within 1 second
        if (now - lastTime < 1000) {
            return;
        }
        recentDispatches.current.set(key, now);

        const id = `${now}-${Math.random().toString(36).substring(2, 7)}`;
        setToasts((prev) => [...prev, { id, type, message }]);

        setTimeout(() => {
            setToasts((prev) => prev.filter((t) => t.id !== id));
        }, 4000);
    }, []);

    const removeToast = (id: string) => {
        setToasts((prev) => prev.filter((t) => t.id !== id));
    };

    const contextValue: ToastContextType = {
        showToast,
        success: (message) => showToast('success', message),
        error: (message) => showToast('error', message),
        warning: (message) => showToast('warning', message),
        info: (message) => showToast('info', message),
    };

    return (
        <ToastContext.Provider value={contextValue}>
            {children}
            {createPortal(
                <div className="toast-container">
                    {toasts.map((toast) => (
                        <div
                            key={toast.id}
                            className={`toast toast-${toast.type}`}
                            onClick={() => removeToast(toast.id)}
                        >
                            <span className="toast-icon">
                                {toast.type === 'success' && '✓'}
                                {toast.type === 'error' && '✕'}
                                {toast.type === 'warning' && '⚠'}
                                {toast.type === 'info' && 'ℹ'}
                            </span>
                            <span className="toast-message">{toast.message}</span>
                        </div>
                    ))}
                </div>,
                document.body
            )}
        </ToastContext.Provider>
    );
}


export function useToast() {
    const context = useContext(ToastContext);
    if (!context) {
        throw new Error('useToast must be used within a ToastProvider');
    }
    return context;
}
