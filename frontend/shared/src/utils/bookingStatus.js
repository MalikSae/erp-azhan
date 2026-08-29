export function getStatusBadgeConfig(status) {
  const config = {
    baru: ['neutral', 'Baru'],
    dp: ['warning', 'DP'],
    lunas: ['success', 'Lunas'],
    batal: ['danger', 'Batal'],
  };
  return config[status] || ['neutral', status];
}

export function getSeatLockIcon(status, isSeatBlocked) {
  if (status === 'batal') return null; // Booking batal tidak menampilkan status seat
  return isSeatBlocked
    ? { icon: 'CircleCheckBig', colorClass: 'text-success-600', label: 'Seat Terkunci' }
    : { icon: 'Loader', colorClass: 'text-warning-600', label: 'Seat Belum Terkunci' };
}
