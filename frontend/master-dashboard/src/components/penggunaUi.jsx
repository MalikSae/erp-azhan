import React from 'react';
import BrandLabel from 'shared/ui/BrandLabel';
import Badge from 'shared/ui/Badge';

// Potongan UI bersama halaman Pengguna.

export const errMsg = (error, fallback) => {
  const m = error?.response?.data?.error;
  return m ? m.charAt(0).toUpperCase() + m.slice(1) : fallback;
};

// Status Aktif/Nonaktif memakai Badge bersama (satu gaya status, lihat Badge.jsx).
export const StatusChip = ({ active, activeLabel = 'Aktif', inactiveLabel = 'Nonaktif' }) =>
  <Badge variant={active ? 'success' : 'neutral'}>{active ? activeLabel : inactiveLabel}</Badge>;

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
