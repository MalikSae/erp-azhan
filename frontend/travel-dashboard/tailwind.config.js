/** @type {import('tailwindcss').Config} */
import sharedPreset from '../shared/tailwind-preset.js';

// Tema UI terpusat di frontend/shared/tailwind-preset.js (identitas SmartHR).
// Jangan menambah palet di sini — ubah preset-nya supaya kedua dashboard seragam.
export default {
  presets: [sharedPreset],
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
    "../shared/src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
