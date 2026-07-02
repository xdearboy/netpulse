import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
    plugins: [react(), tailwindcss()],
    resolve: {
        alias: {
            '@': path.resolve(__dirname, './src'),
        },
    },
    build: {
        outDir: '../cmd/server/static/dist',
        emptyOutDir: true,
    },
    server: {
        proxy: {
            '/api': {
                target: 'http://localhost:8080',
                timeout: 30000,
            },
            '/health': 'http://localhost:8080',
            '/metrics': 'http://localhost:8080',
            '/docs': 'http://localhost:8080',
        },
    },
})
