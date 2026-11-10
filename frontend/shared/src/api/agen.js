import client from './client';

// Keagenan berbasis akun (spek keagenan 2026-11-03). Semua id agen di sini adalah id akun (customer_accounts).

export async function listPengajuanAgen() {
  const { data } = await client.get('/api/admin/agen/pengajuan');
  return data || [];
}

// B1 Keputusan admin atas satu pengajuan (spek bagian 7): satu aksi, satu transaksi.
export const AKSI_KEPUTUSAN_AGEN = {
  SETUJUI: 'setujui',
  BEBASKAN: 'bebaskan',
  MINTA_UNGGAH_ULANG: 'minta_unggah_ulang',
  TOLAK: 'tolak',
  SETUJUI_BAYAR_MENYUSUL: 'setujui_bayar_menyusul',
};

export async function keputusanAgen(akunId, aksi, catatan = '') {
  const { data } = await client.post(`/api/admin/agen/${akunId}/keputusan`, { aksi, catatan });
  return data;
}

// alasan wajib (5 sampai 500 karakter); tercatat di riwayat status agen.
export async function ubahStatusAgen(akunId, status, alasan) {
  const { data } = await client.put(`/api/admin/agen/${akunId}/status`, { status, alasan });
  return data;
}

// Unggah bukti oleh admin atas nama pemohon (pemohon kirim lewat WhatsApp).
export async function uploadBuktiPembayaranAgen(pembayaranId, buktiTransferUrl) {
  const { data } = await client.post(`/api/admin/agen/pembayaran/${pembayaranId}/bukti`, {
    bukti_transfer_url: buktiTransferUrl,
  });
  return data;
}

// Picker agen: agen aktif satu brand. brandId wajib untuk super admin.
export async function listAgenAktif({ brandId, q } = {}) {
  const params = {};
  if (brandId) params.brand_id = brandId;
  if (q) params.q = q;
  const { data } = await client.get('/api/admin/agen/aktif', { params });
  return data || [];
}

// B2 Daftar Agen (aktif & nonaktif) + jumlah_closing & total_komisi.
// brandId hanya dipakai super admin (kosong = semua brand).
export async function listAgen({ q, brandId } = {}) {
  const params = {};
  if (q) params.q = q;
  if (brandId) params.brand_id = brandId;
  const { data } = await client.get('/api/admin/agen', { params });
  return data || [];
}

// Peringkat agen per periode (dari/sampai YYYY-MM-DD, wajib).
export async function getPeringkatAgen({ dari, sampai, brandId }) {
  const params = { dari, sampai };
  if (brandId) params.brand_id = brandId;
  const { data } = await client.get('/api/admin/agen/peringkat', { params });
  return data || [];
}

// B2a Detail Agen (8 kelompok data).
export async function getDetailAgen(akunId) {
  const { data } = await client.get(`/api/admin/agen/${akunId}`);
  return data;
}

// B3 Riwayat Komisi brand. filter: { agen_id, jenis, dari, sampai, limit, offset }.
export async function listRiwayatKomisi(filter = {}) {
  const params = Object.fromEntries(Object.entries(filter).filter(([, v]) => v !== '' && v != null));
  const { data } = await client.get('/api/admin/agen/komisi', { params });
  return data || [];
}

// B5 Pakai Kredit Cashback di detail booking (dimatikan, lihat KOMISI_REPEAT_CASHBACK_AKTIF).
export async function getKreditCashback(bookingId) {
  const { data } = await client.get(`/api/admin/bookings/${bookingId}/cashback`);
  return data;
}

export async function pakaiKreditCashback(bookingId, jamaahId, nominal) {
  const { data } = await client.post(`/api/admin/bookings/${bookingId}/cashback`, { jamaah_id: jamaahId, nominal });
  return data;
}

// Dikirim setelah pengajuan pencairan diputuskan, supaya angka "menunggu" di sidebar ikut berubah.
export const EVENT_PENCAIRAN_KOMISI = 'pencairan-komisi-diputuskan';

// C1 Persetujuan Pencairan (Admin Master, lintas brand).
export async function listPencairanAdmin({ status, brandId } = {}) {
  const params = {};
  if (status) params.status = status;
  if (brandId) params.brand_id = brandId;
  const { data } = await client.get('/api/admin/pencairan', { params });
  return data || [];
}

export async function setujuiPencairan(id, buktiTransferKeluarUrl) {
  const { data } = await client.post(`/api/admin/pencairan/${id}/setujui`, {
    bukti_transfer_keluar_url: buktiTransferKeluarUrl,
  });
  return data;
}

export async function tolakPencairan(id, alasan) {
  const { data } = await client.post(`/api/admin/pencairan/${id}/tolak`, { alasan });
  return data;
}

// C4 Override kaitan (Admin Master): pindahkan kunci jamaah ke agen lain.
export async function cariJamaahKaitan(q, brandId) {
  const params = { q };
  if (brandId) params.brand_id = brandId;
  const { data } = await client.get('/api/admin/kaitan-agen', { params });
  return data || [];
}

export async function listKaitanLog(jamaahId) {
  const { data } = await client.get(`/api/admin/kaitan-agen/${jamaahId}/log`);
  return data || [];
}

// pindahkanKomisi wajib true bila jamaah sudah punya komisi tercatat (mode 'agen').
export async function gantiKaitanAgen(jamaahId, { mode, agenAkunId, alasan, pindahkanKomisi = false }) {
  const { data } = await client.put(`/api/admin/kaitan-agen/${jamaahId}`, {
    mode,
    agen_akun_id: mode === 'agen' ? agenAkunId : null,
    alasan,
    pindahkan_komisi: mode === 'agen' && pindahkanKomisi,
  });
  return data;
}

// C4 Override pada akun: pasang, ganti, atau lepas agen yang mengikat akun.
export async function cariAkunKaitan(q, brandId) {
  const params = { q };
  if (brandId) params.brand_id = brandId;
  const { data } = await client.get('/api/admin/kaitan-agen/akun', { params });
  return data || [];
}

export async function listKaitanAkunLog(akunId) {
  const { data } = await client.get(`/api/admin/kaitan-agen/akun/${akunId}/log`);
  return data || [];
}

export async function gantiAgenAkun(akunId, { agenAkunId, alasan }) {
  const { data } = await client.put(`/api/admin/kaitan-agen/akun/${akunId}`, {
    agen_akun_id: agenAkunId ?? null,
    alasan,
  });
  return data;
}
