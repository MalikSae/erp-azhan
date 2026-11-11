export function getStatusBadgeConfig(status) {
  const config = {
    draft: ['draft', 'Draft'],
    baru: ['neutral', 'Baru'],
    dp: ['warning', 'DP'],
    lunas: ['success', 'Lunas'],
    batal: ['danger', 'Batal'],
  };
  return config[status] || ['neutral', status];
}

export function getSeatLockIcon(status, isSeatBlocked) {
  if (status === 'batal' || status === 'draft') return null; // Booking batal & draft tidak menampilkan status seat
  return isSeatBlocked
    ? { icon: 'CircleCheckBig', colorClass: 'text-success-600', label: 'Seat Di-hold' }
    : { icon: 'Loader', colorClass: 'text-warning-600', label: 'Seat Belum Di-hold' };
}

// Status pembayaran: satu sumber label & gaya untuk Detail Booking dan daftar Pembayaran.
// Warna diatur Badge (satu gaya status); di sini cukup label dan nada.
export const PAYMENT_STATUS_META = {
  pending: { label: 'Menunggu', variant: 'warning' },
  confirmed: { label: 'Terkonfirmasi', variant: 'success' },
  rejected: { label: 'Ditolak', variant: 'danger' },
  // void: pembayaran terkonfirmasi yang dibatalkan admin (salah catat); tidak ikut dihitung sebagai dana masuk.
  void: { label: 'Dibatalkan', variant: 'neutral' },
};

export function getPaymentStatusMeta(status) {
  return PAYMENT_STATUS_META[status] || { label: status || '-', variant: 'neutral' };
}
