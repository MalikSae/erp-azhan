// Format bersama halaman Agen & Komisi (B2, B2a, B3).

export const money = (value) =>
  new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);

export const dateLabel = (value) =>
  !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(value));

export const JENIS_KOMISI = {
  langsung: 'Komisi langsung',
  pembinaan: 'Bonus pembinaan',
  repeat_order: 'Repeat order',
  cashback: 'Cashback',
};

// Ketersediaan per baris ledger (keputusan D4 & D6).
export const KETERSEDIAAN = {
  tersedia: { label: 'Tersedia', variant: 'success' },
  tertahan: { label: 'Tertahan', variant: 'pending' },
  kredit: { label: 'Kredit cashback', variant: 'neutral' },
};

export const STATUS_AGEN = {
  aktif: { label: 'Aktif', variant: 'success' },
  nonaktif: { label: 'Nonaktif', variant: 'neutral' },
};
