import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Port tetap agar tidak tertukar dengan master-dashboard (5173) dan CORS backend.
  server: { port: 5174, strictPort: true },
})
