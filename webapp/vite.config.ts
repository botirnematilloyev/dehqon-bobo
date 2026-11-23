import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

// Dev: /api so'rovlari lokal Go API'ga (HTTP_PORT). Prod'da ilova va API
// bitta domenda (Caddy), shuning uchun klient doim nisbiy `/api/v1`.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // cloudflared tunnel orqali ochilganda Host sarlavhasi boshqa bo'ladi.
    allowedHosts: true,
    proxy: {
      '/api': process.env.API_PROXY ?? 'http://localhost:8087',
    },
  },
  test: {
    environment: 'node',
  },
})
