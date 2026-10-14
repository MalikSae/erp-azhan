import React from 'react';
import Badge from './Badge';

// Standar halaman detail (acuan: Detail Paket).
//
//   <Surface className="grid grid-cols-3 divide-x divide-neutral-200">   ← strip ringkasan
//     <SummaryItem label="…" value="…" hint="…" />
//   </Surface>
//   <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
//     <Surface className="lg:col-span-2 divide-y divide-neutral-200/80">
//       <Section title="…">…<Field label="…">nilai</Field>…</Section>
//     </Surface>
//     <Surface className="divide-y divide-neutral-200/80">…</Surface>
//   </div>
//
// Satu card per kolom, section dipisah garis tipis; tanpa kotak bersarang dan tanpa ikon di judul.

export const Surface = ({ children, className = '' }) => (
  <div className={`bg-white border border-neutral-200 rounded-xl ${className}`}>{children}</div>
);

// Satu section di dalam Surface: heading (+ aside opsional) lalu isi.
export const Section = ({ title, aside, children }) => (
  <section className="p-5 sm:p-6">
    <div className="flex items-center justify-between gap-3 mb-4">
      <h3 className="font-heading text-sm font-bold text-neutral-900 flex items-center gap-2">
        {title}
      </h3>
      {aside}
    </div>
    {children}
  </section>
);

// Pasangan label kecil + nilai.
export const Field = ({ label, children, className = '' }) => (
  <div className={className}>
    <span className="block text-[11px] text-neutral-400 font-body mb-0.5">{label}</span>
    <div className="text-sm font-medium font-body text-neutral-900 break-words">{children}</div>
  </div>
);

// Satu sel pada strip ringkasan.
export const SummaryItem = ({ label, value, hint, onClick }) => {
  const body = (
    <>
      <span className="block text-[11px] text-neutral-400 font-body">{label}</span>
      <span className="block font-heading text-xl font-bold text-neutral-900 mt-0.5 truncate">{value}</span>
      {hint && <span className={`block text-[11px] font-body mt-0.5 ${onClick ? 'text-primary-600' : 'text-neutral-500'}`}>{hint}</span>}
    </>
  );
  if (!onClick) return <div className="px-5 py-4 min-w-0">{body}</div>;
  return (
    <button type="button" onClick={onClick} className="px-5 py-4 min-w-0 text-left cursor-pointer transition-colors hover:bg-neutral-50">
      {body}
    </button>
  );
};

// Pil di header halaman detail: putih bergaris tegas agar terbaca di atas latar abu-abu.
export const PILL_STRONG = '!bg-white !border-neutral-300 !text-neutral-900';

// Brand = pil netral dengan titik warna brand (warna hanya penanda).
export const BrandBadge = ({ name, color }) => (
  <Badge
    variant="neutral"
    className={PILL_STRONG}
    icon={<span className="h-1.5 w-1.5 shrink-0 rounded-full" style={{ backgroundColor: color || '#9CA3AF' }} />}
  >
    {name}
  </Badge>
);

// Tabel: judul (+ aksi) berada DI LUAR kartu; kartu hanya berisi tabel selebar penuh.
export const TableCard = ({ title, aside, children }) => (
  <div>
    <div className="mb-3 flex items-center justify-between gap-3">
      <h3 className="font-heading text-sm font-bold text-neutral-900">{title}</h3>
      {aside}
    </div>
    <Surface className="overflow-hidden">{children}</Surface>
  </div>
);
