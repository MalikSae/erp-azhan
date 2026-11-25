// Format bersama halaman Paket (daftar, detail, form).

export const STATUS_LABEL = { published: 'Published', draft: 'Draft', archived: 'Archived' };

export const STATUS_OPTIONS = Object.entries(STATUS_LABEL).map(([value, label]) => ({ value, label }));

const BULAN = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'];
const BULAN_PENDEK = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];

const keTanggal = (value) => {
  if (!value) return null;
  const d = new Date(value);
  return isNaN(d.getTime()) ? null : d;
};

// "14 September 2026"; teks asli bila tidak terbaca sebagai tanggal.
export const formatTanggal = (value) => {
  if (!value) return '-';
  const d = keTanggal(value);
  return d ? `${d.getDate()} ${BULAN[d.getMonth()]} ${d.getFullYear()}` : value;
};

// "14 Sep 2026".
export const formatTanggalPendek = (value) => {
  const d = keTanggal(value);
  return d ? `${d.getDate()} ${BULAN_PENDEK[d.getMonth()]} ${d.getFullYear()}` : '-';
};

// "09:00" dari "09:00:00".
export const formatJam = (value) => (value ? value.substring(0, 5) : '');

// "Rp 25.000.000"; '-' untuk kosong atau nol.
export const formatRupiah = (amount) => {
  if (amount === undefined || amount === null || amount === '') return '-';
  const num = Number(amount);
  if (isNaN(num) || num <= 0) return '-';
  return 'Rp ' + num.toLocaleString('id-ID');
};

// URL file unggahan (logo, gambar utama) dari path server.
export const assetUrl = (url) => {
  if (!url) return '';
  if (url.startsWith('http') || url.startsWith('blob:')) return url;
  return `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:9090'}${url.startsWith('/') ? '' : '/'}${url}`;
};
