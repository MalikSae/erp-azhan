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
        primary: {
          50: '#FEFDF0',
          100: '#FEF7D6',
          200: '#FDEE9F',
          300: '#FCE366',
          400: '#FBD836',
          500: '#FED853',
          600: '#F5CD3E',
          700: '#E5BD2C',
          800: '#D4AA1E',
          900: '#B89014',
          950: '#8A6A0B',
          DEFAULT: '#FED853',
        },
        neutral: {
          50: '#FAFAFA',
          100: '#F4F4F5',
          200: '#E4E4E7',
          300: '#D4D4D8',
          400: '#A1A1AA',
          500: '#71717A',
          600: '#52525B',
          700: '#3F3F46',
          800: '#27272A',
          900: '#18181B',
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
          bg: '#181C1F',
          surface: '#22272B',
          hover: '#292F34',
          active: '#FED853',
          activeText: '#14171A',
          muted: '#8C95A0',
          border: '#2A3036'
        },
        brand: {
          dark: '#14171A'
        },
        page: {
          bg: '#F6F8FA'
        },
        accent: {
          gold: '#FED853',
          'gold-hover': '#F5CD3E',
          'gold-light': '#FEF7D6',
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
