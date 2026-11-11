import React from 'react';
import BrandLabel from 'shared/ui/BrandLabel';

// Potongan UI bersama halaman Pengguna & Struktur Organisasi (gaya SmartHR).

export const errMsg = (error, fallback) => {
  const m = error?.response?.data?.error;
  return m ? m.charAt(0).toUpperCase() + m.slice(1) : fallback;
};

// Tombol aksi kotak berbingkai 30px.
export const actionBtn =
  'w-8 h-8 flex items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-500 transition-colors disabled:opacity-40 disabled:cursor-not-allowed';

export const StatusChip = ({ active, activeLabel = 'Aktif', inactiveLabel = 'Nonaktif' }) =>
  active ? (
    <span className="inline-flex items-center gap-1.5 text-xs font-body text-success-700">
      <span className="w-2 h-2 rounded-full bg-success-500" />{activeLabel}
    </span>
  ) : (
    <span className="inline-flex items-center gap-1.5 text-xs font-body text-neutral-500">
      <span className="w-2 h-2 rounded-full bg-neutral-300" />{inactiveLabel}
    </span>
  );

// Akses brand: gaya sama dengan penanda brand di daftar Paket (dot warna + nama).
export const AksesChip = ({ brandId, brandName, brandColor }) => {
  const holding = brandId === null || brandId === undefined;
  return (
    <BrandLabel
      name={holding ? 'Holding' : (brandName || `Brand #${brandId}`)}
      color={holding ? '#FF6A00' : brandColor}
    />
  );
};

export const SoftBadge = ({ children, tone = 'primary', title }) => {
  const tones = {
    primary: 'bg-primary-50 text-primary-700 border-primary-200',
    warning: 'bg-warning-50 text-warning-700 border-warning-200',
    neutral: 'bg-neutral-50 text-neutral-600 border-neutral-200',
  };
  return (
    <span title={title} className={`inline-flex items-center px-1.5 py-0.5 rounded-sm text-xs font-medium font-heading border ${tones[tone] || tones.primary}`}>
      {children}
    </span>
  );
};

// Kartu bergaya referensi dengan header judul.
export const Panel = ({ title, right, children, bodyClass = 'p-5' }) => (
  <section className="bg-white rounded-2xl border border-neutral-200">
    {(title || right) && (
      <div className="px-5 py-4 border-b border-neutral-100 flex items-center justify-between gap-3">
        <h3 className="text-base font-heading font-semibold text-neutral-900">{title}</h3>
        {right}
      </div>
    )}
    <div className={bodyClass}>{children}</div>
  </section>
);
