/**
 * Preset Tailwind bersama — identitas UI dashboard internal ERP Azhan.
 * Keputusan 2 Okt 2026: mengikuti referensi SmartHR 100% (palet oranye,
 * latar abu terang, kartu putih, sidebar putih, font Inter). Dibangun ulang
 * sebagai token — tidak menyalin kode/aset template.
 *
 * Dipakai master-dashboard & travel-dashboard lewat `presets: [preset]`.
 * JANGAN duplikasi palet di config masing-masing app lagi.
 */
export default {
  theme: {
    extend: {
      colors: {
        // Primer oranye SmartHR. 500/600 untuk tombol & aksen utama,
        // 50–100 untuk tint latar (menu aktif, badge lembut).
        primary: {
          50: '#FFF4EE',
          100: '#FFE6D9',
          200: '#FFCCB2',
          300: '#FFA880',
          400: '#FB8A52',
          500: '#F26522', // oranye utama (tombol, link aktif)
          600: '#E05510', // hover
          700: '#BA460D',
          800: '#94390F',
          900: '#78300F',
          950: '#411605',
        },
        // Netral abu dingin (gray) khas SmartHR — menggantikan stone hangat.
        neutral: {
          50: '#F9FAFB',
          100: '#F3F4F6',
          200: '#E5E7EB',
          300: '#D1D5DB',
          400: '#9CA3AF',
          500: '#6B7280',
          600: '#4B5563',
          700: '#374151',
          800: '#1F2937',
          900: '#111827',
        },
        success: {
          50: '#f0fdf4', 100: '#dcfce7', 200: '#bbf7d0', 300: '#86efac',
          400: '#4ade80', 500: '#22c55e', 600: '#16a34a', 700: '#15803d',
          800: '#166534', 900: '#14532d',
        },
        warning: {
          50: '#fffbeb', 100: '#fef3c7', 200: '#fde68a', 300: '#fcd34d',
          400: '#fbbf24', 500: '#f59e0b', 600: '#d97706', 700: '#b45309',
          800: '#92400e', 900: '#78350f',
        },
        danger: {
          50: '#fef2f2', 100: '#fee2e2', 200: '#fecaca', 300: '#fca5a5',
          400: '#f87171', 500: '#ef4444', 600: '#dc2626', 700: '#b91c1c',
          800: '#991b1b', 900: '#7f1d1d',
        },
        // Aksen sekunder untuk stat tile & chart (ungu/biru/teal ala referensi).
        info: {
          50: '#eff6ff', 100: '#dbeafe', 500: '#3b82f6', 600: '#2563eb',
        },
        violet: {
          50: '#f5f3ff', 100: '#ede9fe', 500: '#8b5cf6', 600: '#7c3aed',
        },
        // Sidebar SmartHR: putih, teks abu, item aktif tint oranye.
        sidebar: {
          bg: '#FFFFFF',
          surface: '#F9FAFB',
          hover: '#F3F4F6',
          active: 'rgba(242, 101, 34, 0.10)',
          activeText: '#F26522',
          muted: '#6B7280',
          border: '#E5E7EB',
        },
        brand: {
          dark: '#1F2937',
          ivory: '#F7F7F9', // nama lama dipertahankan agar kelas lama tidak pecah
        },
        page: {
          bg: '#F7F7F9',
        },
        // Nama lama `accent-gold` masih dipakai beberapa halaman — dipetakan
        // ke oranye supaya tidak ada kelas yang pecah; bersihkan bertahap.
        accent: {
          gold: '#F26522',
          'gold-hover': '#E05510',
          'gold-light': '#FFE6D9',
        },
      },
      borderRadius: {
        lg: '0.5rem',    // 8px: kontrol kompak
        xl: '0.625rem',  // 10px: tombol & field
        '2xl': '0.75rem',// 12px: kartu
        '3xl': '1rem',   // 16px: panel besar
      },
      boxShadow: {
        card: '0 1px 2px 0 rgba(16, 24, 40, 0.04), 0 1px 3px 0 rgba(16, 24, 40, 0.06)',
        'card-hover': '0 12px 16px -4px rgba(16, 24, 40, 0.08), 0 4px 6px -2px rgba(16, 24, 40, 0.03)',
      },
      fontFamily: {
        heading: ['Inter', 'system-ui', 'sans-serif'],
        body: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
};
