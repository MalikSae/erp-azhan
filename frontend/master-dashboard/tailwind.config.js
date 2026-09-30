/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
    "../shared/src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Palet holding Azhan Grup: onyx + champagne gold di atas gading.
        // Emas hanya aksen (menu aktif, logo, garis tipis); tombol utama memakai onyx.
        // 50–500: latar/isi dengan teks onyx di atasnya. 600–950: teks/ikon di latar terang.
        primary: {
          50: '#FAF7EF',
          100: '#F3ECDC',
          200: '#E8DBBD',
          300: '#D9C392',
          400: '#CFB277',
          500: '#C4A062', // champagne gold (7.2:1 di atas onyx)
          600: '#8C6D2F', // teks/ikon (4.8:1 vs putih), isi dengan teks putih
          700: '#735826', // emas tua: teks & link (6.7:1)
          800: '#5E471C',
          900: '#453414',
          950: '#2C210C',
        },
        // Netral hangat (stone) agar serasi dengan latar gading.
        neutral: {
          50: '#FAFAF9',
          100: '#F5F5F4',
          200: '#E7E5E4',
          300: '#D6D3D1',
          400: '#A8A29E',
          500: '#78716C',
          600: '#57534E',
          700: '#44403C',
          800: '#292524',
          900: '#1C1917',
        },
        success: {
          50: '#f0fdf4',
          100: '#dcfce7',
          200: '#bbf7d0',
          300: '#86efac',
          400: '#4ade80',
          500: '#22c55e',
          600: '#16a34a',
          700: '#15803d',
          800: '#166534',
          900: '#14532d',
        },
        warning: {
          50: '#fffbeb',
          100: '#fef3c7',
          200: '#fde68a',
          300: '#fcd34d',
          400: '#fbbf24',
          500: '#f59e0b',
          600: '#d97706',
          700: '#b45309',
          800: '#92400e',
          900: '#78350f',
        },
        danger: {
          50: '#fef2f2',
          100: '#fee2e2',
          200: '#fecaca',
          300: '#fca5a5',
          400: '#f87171',
          500: '#ef4444',
          600: '#dc2626',
          700: '#b91c1c',
          800: '#991b1b',
          900: '#7f1d1d',
        },
        sidebar: {
          bg: '#16181B',
          surface: '#1F2226',
          hover: '#26292E',
          active: 'rgba(196, 160, 98, 0.14)', // tint champagne
          activeText: '#D9C392',
          muted: '#A8A49C',
          border: '#2A2D31'
        },
        brand: {
          dark: '#16181B', // onyx
          ivory: '#F7F5F0' // gading
        },
        page: {
          bg: '#F7F5F0'
        },
        accent: {
          gold: '#C4A062',
          'gold-hover': '#8C6D2F',
          'gold-light': '#F3ECDC',
        }
      },
      borderRadius: {
        'xl': '0.75rem', // 12px
        '2xl': '1rem',   // 16px
        '3xl': '1.5rem', // 24px
      },
      boxShadow: {
        'card': '0 1px 3px 0 rgba(0, 0, 0, 0.04), 0 1px 2px -1px rgba(0, 0, 0, 0.04)',
        'card-hover': '0 10px 15px -3px rgba(0, 0, 0, 0.05), 0 4px 6px -4px rgba(0, 0, 0, 0.05)',
      },
      fontFamily: {
        heading: ["DM Sans", "sans-serif"],
        body: ["DM Sans", "sans-serif"],
      }
    },
  },
  plugins: [],
}
