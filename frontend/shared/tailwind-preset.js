/**
 * Preset Tailwind bersama — identitas UI dashboard internal ERP Azhan.
 * Keputusan 11 Nov 2026: gaya dash.cloudflare.com (lihat DESIGN.md di root
 * repo): hampir monokrom, latar putih, garis tipis, radius 6–8px.
 * `primary` sengaja skala abu-hitam supaya semua aksen lama ikut monokrom.
 * Oranye hanya lewat `accent`: tombol primer (gradien), logo, promo, grafik.
 * Navigasi, tab, link, dan fokus tetap monokrom.
 *
 * Dipakai master-dashboard lewat `presets: [preset]`.
 * JANGAN duplikasi palet di config masing-masing app lagi.
 */
export default {
  theme: {
    extend: {
      colors: {
        // Primer monokrom. 50–100 latar terpilih/aktif, 500 ikon & fokus,
        // 600 isi tombol primer (teks putih 15:1), 700 hover & teks link.
        primary: {
          50: '#F7F7F7',
          100: '#F0F0F0',
          200: '#E3E3E3',
          300: '#CFCFCF',
          400: '#9E9E9E',
          500: '#404040',
          600: '#262626',
          700: '#171717',
          800: '#0F0F0F',
          900: '#0A0A0A',
          950: '#000000',
        },
        // Oranye Cloudflare (diukur dari tombol "Watch now" dash.cloudflare.com):
        // tombol primer, logo, promo, seri grafik utama. Bukan navigasi.
        accent: {
          50: '#FFF4EE',
          100: '#FFE4D6',
          400: '#FF8130', // atas gradien tombol
          500: '#FF6A00', // bawah gradien tombol, logo
          600: '#E25A00', // border tombol
          700: '#C93F08', // teks oranye di latar putih (5.00:1)
        },
        // Netral abu murni (Tailwind neutral), mengikuti Cloudflare #262626.
        neutral: {
          50: '#FAFAFA',
          100: '#F5F5F5',
          200: '#E5E5E5',
          300: '#D4D4D4',
          400: '#A3A3A3',
          500: '#737373',
          600: '#595959',
          700: '#404040',
          800: '#262626',
          900: '#171717',
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
        // Warna data (seri grafik). Bukan untuk tombol atau navigasi.
        info: {
          50: '#eff6ff', 100: '#dbeafe', 500: '#3b82f6', 600: '#2563eb',
        },
        violet: {
          50: '#f5f3ff', 100: '#ede9fe', 500: '#8b5cf6', 600: '#7c3aed',
        },
        // Sidebar putih; item aktif latar abu, teks hitam, tanpa garis warna.
        sidebar: {
          bg: '#FFFFFF',
          surface: '#FAFAFA',
          hover: '#F5F5F5',
          active: '#F0F0F0',
          activeText: '#171717',
          muted: '#404040',
          label: '#737373',
          border: '#E5E5E5',
        },
        page: {
          bg: '#FFFFFF',
        },
      },
      // Radius mengikuti dash.cloudflare.com. Nilai juga dikunci di
      // src/corner-radius.css supaya build dev dan prod seragam.
      borderRadius: {
        sm: '4px',       // badge, chip
        md: '6px',       // tombol, field, item menu
        lg: '6px',
        xl: '6px',
        '2xl': '8px',    // kartu, tabel, kolom cari besar
        '3xl': '8px',    // modal, popover, dropdown
      },
      // Datar: kartu tanpa bayangan, batas lewat border 1px.
      // Bayangan hanya untuk lapisan mengambang.
      boxShadow: {
        card: 'none',
        'card-hover': 'none',
        popover: '0 4px 12px rgba(0, 0, 0, 0.08), 0 1px 2px rgba(0, 0, 0, 0.04)',
        modal: '0 16px 48px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.04)',
        // Tombol gaya Cloudflare: kilap tipis di tepi atas + bayangan lembut.
        'btn-primary': 'inset 0 1px 0 rgba(255, 255, 255, 0.28), 0 2px 6px -1px rgba(255, 106, 0, 0.30)',
        'btn-secondary': '0 1px 2px rgba(0, 0, 0, 0.05)',
      },
      backgroundImage: {
        'btn-primary': 'linear-gradient(180deg, #FF8130 0%, #FF6A00 100%)',
        'btn-primary-hover': 'linear-gradient(180deg, #FF7520 0%, #F25F00 100%)',
      },
      transitionTimingFunction: {
        cf: 'cubic-bezier(0.19, 1, 0.22, 1)',
      },
      // Inter untuk judul, angka statistik, tombol, badge (font-heading);
      // Roboto untuk teks isi: body, tabel, field, menu (font-body).
      fontFamily: {
        heading: ['Inter', 'Roboto', 'system-ui', 'sans-serif'],
        body: ['Roboto', 'system-ui', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Consolas', 'monospace'],
      },
    },
  },
};
