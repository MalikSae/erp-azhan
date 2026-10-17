// Warna badge kategori paket: hex #rrggbb dari color picker (kosong = otomatis, dipilihkan situs booking).

export const BADGE_FALLBACK = '#9ca3af';

// Pilihan cepat di bawah color picker (sama dengan warna otomatis situs booking, plus merah dan hijau).
export const BADGE_PRESETS = [
  { label: 'Hijau tosca', hex: '#0f766e' },
  { label: 'Biru', hex: '#2563eb' },
  { label: 'Ungu', hex: '#7c3aed' },
  { label: 'Cokelat emas', hex: '#a16207' },
  { label: 'Merah muda', hex: '#db2777' },
  { label: 'Abu gelap', hex: '#475569' },
  { label: 'Merah', hex: '#b91c1c' },
  { label: 'Hijau', hex: '#15803d' },
];

export const isBadgeHex = (v) => /^#[0-9a-f]{6}$/i.test(v || '');

// Teks putih atau gelap, mana yang kontrasnya lebih baik di atas warna latar (luminans relatif WCAG).
export function badgeTextColor(hex) {
  if (!isBadgeHex(hex)) return '#ffffff';
  const [r, g, b] = [1, 3, 5].map((i) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  const lum = 0.2126 * r + 0.7152 * g + 0.0722 * b;
  return lum > 0.4 ? '#1f2937' : '#ffffff';
}
