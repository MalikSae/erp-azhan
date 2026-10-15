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
    ? { icon: 'CircleCheckBig', colorClass: 'text-success-600', label: 'Seat Terkunci' }
    : { icon: 'Loader', colorClass: 'text-warning-600', label: 'Seat Belum Terkunci' };
}

// Status pembayaran: satu sumber label & gaya untuk Detail Booking dan daftar Pembayaran.
// className mempertegas warna agar terbaca di atas latar halaman abu-abu.
export const PAYMENT_STATUS_META = {
  pending: { label: 'Menunggu', variant: 'pending', className: '!bg-warning-100 !border-warning-300 !text-warning-800' },
  confirmed: { label: 'Terkonfirmasi', variant: 'success', className: '!bg-success-100 !border-success-300 !text-success-800' },
  rejected: { label: 'Ditolak', variant: 'danger', className: '!bg-danger-100 !border-danger-200 !text-danger-700' },
};

export function getPaymentStatusMeta(status) {
  return PAYMENT_STATUS_META[status] || { label: status || '-', variant: 'neutral', className: '' };
}
