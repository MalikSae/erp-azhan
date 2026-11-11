import React from 'react';

// Penanda brand di tabel/daftar: titik warna brand + nama, tanpa kotak.
// Sama dengan penanda brand di daftar Paket. Warna brand hanya penanda, bukan warna UI.
const BrandLabel = ({ name, color, className = '' }) => (
  <span className={`inline-flex items-center gap-1.5 font-body text-sm font-medium text-neutral-700 whitespace-nowrap ${className}`}>
    <span className="h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: color || '#A3A3A3' }} />
    {name}
  </span>
);

export default BrandLabel;
