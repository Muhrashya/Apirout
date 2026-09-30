import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
    plugins: [react()],
    resolve: {
        alias: {
            '@': path.resolve(__dirname, './src'),
        },
    },
    server: {
        port: 3000,
        proxy: {
            '/auth': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/client': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/api': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/users': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/groups': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/connections': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/queries': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/endpoints': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/api-clients': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/dashboard': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/audit-logs': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/openapi.json': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/postman.json': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
            '/health': {
                target: 'http://localhost:8080',
                changeOrigin: true,
            },
        },
    },
    build: {
        outDir: 'dist',
        emptyOutDir: true,
    },
})
