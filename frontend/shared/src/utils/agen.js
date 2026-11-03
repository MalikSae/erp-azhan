// Format bersama halaman Agen & Komisi (B2, B2a, B3).

export const money = (value) =>
  new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(Number(value) || 0);

export const dateLabel = (value) =>
  !value ? '-' : new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(value));

// Cashback dan komisi repeat order dimatikan di backend (komisi.KomisiRepeatCashbackAktif = false,
// spek keagenan K11/K12). Ubah flag ini bersamaan dengan flag backend bila fitur dihidupkan lagi.
export const KOMISI_REPEAT_CASHBACK_AKTIF = false;

// Label lengkap tetap ada supaya baris ledger lama masih terbaca; filter hanya menawarkan jenis aktif.
export const JENIS_KOMISI = {
  langsung: 'Komisi langsung',
  pembinaan: 'Bonus pembinaan',
  repeat_order: 'Repeat order',
  cashback: 'Cashback',
};

export const JENIS_KOMISI_FILTER = Object.fromEntries(
  Object.entries(JENIS_KOMISI).filter(([jenis]) => KOMISI_REPEAT_CASHBACK_AKTIF || (jenis !== 'repeat_order' && jenis !== 'cashback')),
);

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
