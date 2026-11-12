import React from 'react';

// EmptyState — satu-satunya tampilan "data kosong" (design-system.md §5.3).
//   variant="list"    (default) daftar/tabel kosong: teks di tengah, lega. Dipakai Table/DataTable.
//   variant="section" bagian kartu kosong (mis. "Belum ada pembayaran"): teks rata kiri, padding kartu.
//   variant="inline"  di dalam wadah yang sudah berpadding (Section, MetaBox): teks saja, tanpa padding.
// Pesan berupa kalimat biasa; boleh ditambah petunjuk aksi ("Catat pembayaran dari jamaah di sini.").
// Tanpa ilustrasi, ikon, atau kotak putus-putus.
const EmptyState = ({ message = 'Data tidak ditemukan', variant = 'list', className = '' }) => {
  if (variant === 'inline') {
    return <p className={`text-sm font-body text-neutral-500 ${className}`}>{message}</p>;
  }
  if (variant === 'section') {
    return <p className={`px-5 py-5 text-sm font-body text-neutral-500 ${className}`}>{message}</p>;
  }
  return (
    <div className={`flex items-center justify-center px-4 py-12 text-center ${className}`}>
      <p className="text-sm font-body text-neutral-500">{message}</p>
    </div>
  );
};

export default EmptyState;
