import React from 'react';

// Nama baris yang bisa diklik di tabel master (design-system.md §3). Tanpa izin ubah, tampil sebagai teks biasa.
const NamaBarisButton = ({ children, onClick, disabled = false }) => (
  <button
    type="button"
    disabled={disabled}
    onClick={onClick}
    className="font-body font-medium text-neutral-900 hover:text-primary-600 disabled:hover:text-neutral-900 disabled:cursor-default text-left leading-snug transition-colors"
  >
    {children}
  </button>
);

export default NamaBarisButton;
