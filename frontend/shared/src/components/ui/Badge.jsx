import React from 'react';

// Badge — SATU-SATUNYA gaya status di dashboard (DESIGN.md, design-system.md §5.2):
// kotak tint tipis, radius 4px, teks xs medium, tanpa ikon dan tanpa titik.
// Halaman cukup memetakan statusnya ke salah satu nada di bawah; jangan
// menimpa warna lewat className (`!bg-…`) atau membuat span status sendiri.
//
//   success  → selesai/baik: Published, Aktif, Lunas, Disetujui, Tayang
//   warning  → perlu tindakan/menunggu: Draft, DP, Menunggu, Mengajukan, Akan tayang
//   danger   → gagal/dibatalkan: Batal, Ditolak, Void
//   info     → informasi netral berwarna: Langsung tersimpan, Direct
//   neutral  → tidak aktif/selesai tanpa makna: Archived, Nonaktif, Berakhir
//   promo    → label promo (oranye accent)
const TONES = {
  success: 'bg-success-50 text-success-700 border-success-200',
  warning: 'bg-warning-50 text-warning-800 border-warning-200',
  danger: 'bg-danger-50 text-danger-700 border-danger-200',
  info: 'bg-info-50 text-info-600 border-info-100',
  neutral: 'bg-neutral-100 text-neutral-700 border-neutral-200',
  promo: 'bg-accent-50 text-accent-700 border-accent-100',
};

// Nama lama tetap diterima supaya pemanggil lama tidak rusak.
const ALIASES = {
  published: 'success', approved: 'success',
  pending: 'warning', draft: 'warning',
  failed: 'danger',
  archived: 'neutral', primary: 'neutral',
};

// `icon` hanya untuk atribut non-status (mis. titik warna brand di BrandBadge).
// `showIcon`/`hideIcon` diabaikan; dipertahankan agar pemanggil lama tetap jalan.
// eslint-disable-next-line no-unused-vars
const Badge = ({ variant = 'neutral', icon, showIcon, hideIcon, children, className = '' }) => {
  const tone = TONES[variant] ? variant : (ALIASES[variant] || 'neutral');
  return (
    <span className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded-sm border text-xs font-medium font-heading whitespace-nowrap ${TONES[tone]} ${className}`}>
      {icon}
      <span>{children}</span>
    </span>
  );
};

// Peta lama berbasis kelas titik (mis. `dot: 'bg-success-500'`) ke nada Badge,
// supaya tabel status lama bisa langsung dirender sebagai Badge.
export const toneFromClass = (cls = '') => {
  if (cls.includes('success')) return 'success';
  if (cls.includes('warning')) return 'warning';
  if (cls.includes('danger')) return 'danger';
  if (cls.includes('info')) return 'info';
  return 'neutral';
};

export default Badge;
