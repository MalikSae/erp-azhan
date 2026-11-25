// Daftar centang izin di form Pengguna (docs/spek/SPEK-AKSES-PENGGUNA-2026-11-25.md §4).
// `brand` menandai kolom yang boleh dicentang untuk pengguna terikat brand; backend
// menegakkan batas yang sama di internal/akses (izinBrand). Ubah keduanya bersamaan.

export const SUPER_ADMIN = 'super_admin';
export const TIM = 'tim';

export const AKSES_LABEL = { [SUPER_ADMIN]: 'Super Admin', [TIM]: 'Tim' };

// Satu baris per menu. lihat/kelola: kode izin yang dicentang bersama.
export const BARIS_IZIN = [
  { id: 'jamaah', label: 'Jamaah', lihat: ['jamaah.view', 'dokumen.view'], kelola: ['jamaah.edit', 'dokumen.edit'], brand: { lihat: true, kelola: true } },
  { id: 'booking', label: 'Booking', lihat: ['booking.view'], kelola: ['booking.edit'], brand: { lihat: true, kelola: true } },
  { id: 'pembayaran', label: 'Pembayaran', lihat: ['payment.view', 'refund.view'], kelola: ['payment.edit', 'refund.edit'], brand: { lihat: true, kelola: true } },
  { id: 'akun', label: 'Akun', lihat: ['akun.view'], kelola: ['akun.edit'], brand: { lihat: true, kelola: true } },
  { id: 'agen', label: 'Agen', lihat: ['agen.view', 'komisi.view'], kelola: ['agen.edit'], brand: { lihat: true, kelola: true } },
  { id: 'paket', label: 'Paket', lihat: ['schedule.view'], kelola: ['schedule.edit'], brand: { lihat: true, kelola: false } },
  { id: 'manifest', label: 'Manifest', lihat: [], kelola: ['manifest.manage'], brand: { lihat: false, kelola: true } },
  { id: 'manasik', label: 'Manasik', lihat: ['manasik.view'], kelola: ['manasik.edit'], brand: { lihat: true, kelola: true } },
  { id: 'masterdata', label: 'Master data', lihat: ['masterdata.view'], kelola: ['masterdata.edit'], brand: { lihat: true, kelola: false } },
  { id: 'perlengkapan', label: 'Perlengkapan', lihat: ['perlengkapan.view'], kelola: ['perlengkapan.edit'], brand: { lihat: true, kelola: true } },
  { id: 'rekening', label: 'Rekening bank', lihat: ['bankaccount.view'], kelola: ['bankaccount.edit'], brand: { lihat: true, kelola: true } },
  { id: 'brand', label: 'Brand', lihat: ['brand.view'], kelola: [], brand: { lihat: true, kelola: false } },
  { id: 'pengumuman', label: 'Pengumuman', lihat: [], kelola: ['pengumuman.manage'], brand: { lihat: false, kelola: true } },
  { id: 'banner', label: 'Banner promosi', lihat: [], kelola: ['banner.manage'], brand: { lihat: false, kelola: true } },
  { id: 'konten', label: 'Konten web', lihat: ['cms.view'], kelola: ['cms.edit', 'cms.blog.write'], brand: { lihat: false, kelola: false } },
  { id: 'haramain', label: 'Jelajah Haramain', lihat: [], kelola: ['haramain.manage'], brand: { lihat: false, kelola: false } },
  { id: 'analitik', label: 'Analitik', lihat: ['report.analytics.view'], kelola: [], brand: { lihat: false, kelola: false } },
];

// Aksi berisiko, dicentang terpisah.
export const IZIN_KHUSUS = [
  { id: 'verifikasi', label: 'Verifikasi pembayaran', kode: ['payment.verify'], brand: true },
  { id: 'void', label: 'Void pembayaran', kode: ['payment.void'], brand: true },
  { id: 'setujui-agen', label: 'Setujui pengajuan agen', kode: ['agen.approve'], brand: true },
  { id: 'setujui-jamaah', label: 'Setujui perubahan data jamaah', kode: ['jamaah.perubahan.approve'], brand: true },
  { id: 'diskon', label: 'Diskon besar', kode: ['booking.diskon.besar'], brand: false },
  { id: 'cairkan', label: 'Cairkan komisi', kode: ['agen.payout.approve'], brand: false },
  { id: 'kaitan', label: 'Ganti kaitan agen', kode: ['agen.kaitan.edit'], brand: false },
  { id: 'terbit', label: 'Terbitkan konten', kode: ['cms.publish', 'cms.blog.publish', 'cms.schema.approve', 'cms.tracking.manage'], brand: false },
];

// Pengisi cepat centang; tidak tersimpan pada pengguna.
export const TEMPLAT_AKSES = [
  {
    id: 'admin-brand',
    label: 'Admin Brand',
    brand: true,
    kode: [
      'jamaah.view', 'dokumen.view', 'jamaah.edit', 'dokumen.edit',
      'booking.view', 'booking.edit',
      'payment.view', 'refund.view', 'payment.edit', 'refund.edit',
      'akun.view', 'akun.edit',
      'agen.view', 'komisi.view', 'agen.edit',
      'schedule.view', 'manifest.manage',
      'manasik.view', 'manasik.edit',
      'masterdata.view',
      'perlengkapan.view', 'perlengkapan.edit',
      'bankaccount.view', 'brand.view',
      'pengumuman.manage', 'banner.manage',
      'payment.verify', 'agen.approve', 'jamaah.perubahan.approve',
    ],
  },
  {
    id: 'keuangan',
    label: 'Keuangan',
    brand: true,
    kode: [
      'booking.view',
      'payment.view', 'refund.view', 'payment.edit', 'refund.edit',
      'schedule.view', 'perlengkapan.view',
      'bankaccount.view', 'bankaccount.edit',
      'payment.verify', 'payment.void',
    ],
  },
  {
    id: 'konten',
    label: 'Konten',
    brand: false,
    kode: ['cms.view', 'cms.edit', 'cms.blog.write', 'pengumuman.manage', 'banner.manage', 'haramain.manage'],
  },
];

// Kode yang boleh dimiliki pengguna dengan akses tertentu ('holding' | 'brand').
export const kodeBoleh = (access) => {
  const brand = access === 'brand';
  const out = new Set();
  BARIS_IZIN.forEach((b) => {
    if (!brand || b.brand.lihat) b.lihat.forEach((k) => out.add(k));
    if (!brand || b.brand.kelola) b.kelola.forEach((k) => out.add(k));
  });
  IZIN_KHUSUS.forEach((k) => { if (!brand || k.brand) k.kode.forEach((x) => out.add(x)); });
  return out;
};

// Rapikan pilihan sebelum simpan: satu kode dari sebuah kelompok berarti seluruh
// kelompoknya, Kelola menyertakan Lihat, dan kode di luar batas akses dibuang.
export const rapikanIzin = (kode, access) => {
  const punya = new Set(kode);
  const boleh = kodeBoleh(access);
  const out = new Set();
  const tambah = (list) => list.forEach((k) => { if (boleh.has(k)) out.add(k); });
  BARIS_IZIN.forEach((b) => {
    const kelola = b.kelola.some((k) => punya.has(k) && boleh.has(k));
    const lihat = kelola || b.lihat.some((k) => punya.has(k) && boleh.has(k));
    if (lihat) tambah(b.lihat);
    if (kelola) tambah(b.kelola);
  });
  IZIN_KHUSUS.forEach((k) => { if (k.kode.some((x) => punya.has(x) && boleh.has(x))) tambah(k.kode); });
  return [...out].sort();
};
