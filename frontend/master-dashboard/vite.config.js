import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Port tetap agar tidak tertukar dengan travel-dashboard (5174) dan CORS backend.
  server: { port: 5173, strictPort: true },
})
