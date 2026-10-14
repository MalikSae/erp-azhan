import React from 'react';

// Potongan UI bersama halaman Pengguna & Struktur Organisasi (gaya SmartHR).

export const errMsg = (error, fallback) => {
  const m = error?.response?.data?.error;
  return m ? m.charAt(0).toUpperCase() + m.slice(1) : fallback;
};

// Tombol aksi kotak berbingkai 30px.
export const actionBtn =
  'w-[30px] h-[30px] flex items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-500 transition-colors disabled:opacity-40 disabled:cursor-not-allowed';

export const StatusChip = ({ active, activeLabel = 'Aktif', inactiveLabel = 'Nonaktif' }) =>
  active ? (
    <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-success-700">
      <span className="w-2 h-2 rounded-full bg-success-500" />{activeLabel}
    </span>
  ) : (
    <span className="inline-flex items-center gap-1.5 text-[12px] font-body text-neutral-500">
      <span className="w-2 h-2 rounded-full bg-neutral-300" />{inactiveLabel}
    </span>
  );

// Chip akses brand: dot warna + teks gelap.
export const AksesChip = ({ brandId, brandName, brandColor }) => {
  const holding = brandId === null || brandId === undefined;
  return (
    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[12px] font-medium font-body bg-white border border-neutral-200 text-neutral-700 whitespace-nowrap">
      <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: holding ? '#F26522' : (brandColor || '#9CA3AF') }} />
      {holding ? 'Holding' : (brandName || `Brand #${brandId}`)}
    </span>
  );
};

export const SoftBadge = ({ children, tone = 'primary', title }) => {
  const tones = {
    primary: 'bg-primary-50 text-primary-700 border-primary-200',
    warning: 'bg-warning-50 text-warning-700 border-warning-200',
    neutral: 'bg-neutral-50 text-neutral-600 border-neutral-200',
  };
  return (
    <span title={title} className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium font-body border ${tones[tone] || tones.primary}`}>
      {children}
    </span>
  );
};

// Kartu bergaya referensi dengan header judul.
export const Panel = ({ title, right, children, bodyClass = 'p-5' }) => (
  <section className="bg-white rounded-2xl border border-neutral-200 shadow-card">
    {(title || right) && (
      <div className="px-5 py-4 border-b border-neutral-100 flex items-center justify-between gap-3">
        <h3 className="text-[15px] font-heading font-bold text-neutral-900">{title}</h3>
        {right}
      </div>
    )}
    <div className={bodyClass}>{children}</div>
  </section>
);
