import React from 'react';

// LoadingSpinner — satu-satunya tanda memuat (design-system.md §5.3).
//   variant="inline" (default) di dalam wadah yang sudah ada (tabel, kartu, daftar).
//   variant="card"   pengganti satu blok yang belum termuat: kartu putih standar berisi spinner.
// Jangan menulis teks "Memuat…" sendiri atau membungkus spinner dengan kartu buatan sendiri.
const LoadingSpinner = ({ variant = 'inline', label = 'Memuat' }) => {
  const spinner = (
    <div className="flex justify-center items-center py-8" role="status" aria-label={label}>
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
    </div>
  );
  if (variant === 'card') {
    return <div className="bg-white border border-neutral-200 rounded-2xl p-5">{spinner}</div>;
  }
  return spinner;
};

export default LoadingSpinner;
