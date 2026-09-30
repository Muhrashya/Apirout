import React, { createContext, useContext, useState, useEffect } from 'react';
import { apiClient } from '../api/apiClient';

export interface BrandingConfig {
    app_name: string;
    tagline: string;
    logo_url: string;
    favicon_url: string;
    browser_title: string;
}

const DEFAULT_BRANDING: BrandingConfig = {
    app_name: 'Apirout',
    tagline: 'API GATEWAY',
    logo_url: '/logo.png',
    favicon_url: '/favicon.png',
    browser_title: 'Apirout Console',
};

const STORAGE_KEY = 'apirout_branding_config';

interface BrandingContextType {
    branding: BrandingConfig;
    isLoading: boolean;
    updateBranding: (newConfig: Partial<BrandingConfig>) => Promise<void>;
    resetBranding: () => Promise<void>;
}

const BrandingContext = createContext<BrandingContextType | undefined>(undefined);

// Helper to update the browser favicon dynamically
function applyFavicon(faviconUrl: string) {
    try {
        let link: HTMLLinkElement | null = document.querySelector("link[rel*='icon']");
        if (!link) {
            link = document.createElement('link');
            link.rel = 'icon';
            document.head.appendChild(link);
        }
        link.type = faviconUrl.endsWith('.png') ? 'image/png' : 'image/x-icon';
        link.href = faviconUrl;
    } catch {
        // Ignore in non-browser envs
    }
}

// Helper to update the browser document title
function applyBrowserTitle(title: string) {
    try {
        document.title = title;
    } catch {
        // Ignore in non-browser envs
    }
}

export function BrandingProvider({ children }: { children: React.ReactNode }) {
    const [branding, setBranding] = useState<BrandingConfig>(() => {
        try {
            const cached = localStorage.getItem(STORAGE_KEY);
            if (cached) {
                const parsed = JSON.parse(cached);
                return { ...DEFAULT_BRANDING, ...parsed };
            }
        } catch {
            // fallback
        }
        return DEFAULT_BRANDING;
    });
    const [isLoading, setIsLoading] = useState(false);

    // Apply branding to DOM immediately on initial state
    useEffect(() => {
        applyFavicon(branding.favicon_url || branding.logo_url || '/favicon.png');
        applyBrowserTitle(branding.browser_title || `${branding.app_name} Console`);
    }, [branding]);

    // Fetch live branding from backend API on mount
    useEffect(() => {
        let isMounted = true;
        const fetchBranding = async () => {
            try {
                setIsLoading(true);
                const res = await apiClient.get<any>('/branding');
                const data = res?.data || res;
                if (data && isMounted && typeof data === 'object') {
                    const merged: BrandingConfig = {
                        app_name: data.app_name || DEFAULT_BRANDING.app_name,
                        tagline: data.tagline || DEFAULT_BRANDING.tagline,
                        logo_url: data.logo_url || DEFAULT_BRANDING.logo_url,
                        favicon_url: data.favicon_url || DEFAULT_BRANDING.favicon_url,
                        browser_title: data.browser_title || DEFAULT_BRANDING.browser_title,
                    };
                    setBranding(merged);
                    try {
                        localStorage.setItem(STORAGE_KEY, JSON.stringify(merged));
                    } catch {
                        // ignore storage error
                    }
                }
            } catch {
                // If API fails (e.g. offline), use cached/default
            } finally {
                if (isMounted) setIsLoading(false);
            }
        };

        fetchBranding();
        return () => {
            isMounted = false;
        };
    }, []);

    const updateBranding = async (newConfig: Partial<BrandingConfig>) => {
        const merged: BrandingConfig = {
            ...branding,
            ...newConfig,
        };
        // Optimistic UI update
        setBranding(merged);
        try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify(merged));
        } catch {
            // ignore
        }
        applyFavicon(merged.favicon_url || merged.logo_url);
        applyBrowserTitle(merged.browser_title);

        // Sync with backend
        try {
            await apiClient.put('/branding', merged);
        } catch (err) {
            console.error('Failed to sync branding with server:', err);
            throw err;
        }
    };

    const resetBranding = async () => {
        setBranding(DEFAULT_BRANDING);
        try {
            localStorage.removeItem(STORAGE_KEY);
        } catch {
            // ignore
        }
        applyFavicon(DEFAULT_BRANDING.favicon_url);
        applyBrowserTitle(DEFAULT_BRANDING.browser_title);

        try {
            await apiClient.put('/branding', DEFAULT_BRANDING);
        } catch (err) {
            console.error('Failed to reset branding on server:', err);
            throw err;
        }
    };

    return (
        <BrandingContext.Provider value={{ branding, isLoading, updateBranding, resetBranding }}>
            {children}
        </BrandingContext.Provider>
    );
}

export function useBranding() {
    const context = useContext(BrandingContext);
    if (!context) {
        throw new Error('useBranding must be used within a BrandingProvider');
    }
    return context;
}
