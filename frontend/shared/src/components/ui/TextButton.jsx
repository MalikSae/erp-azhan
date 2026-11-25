import React from 'react';

// Aksi kedua yang tampil sebagai tautan teks (design-system.md §3: link teks bergaris
// bawah). Dipakai bila aksi itu bukan tombol utama kartu, mis. "Ganti kata sandi".
const TextButton = ({ children, onClick, type = 'button', className = '' }) => (
  <button
    type={type}
    onClick={onClick}
    className={`font-body text-sm font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900 ${className}`}
  >
    {children}
  </button>
);

export default TextButton;
