import client from './client';

// Agen Syiar — Persetujuan Agen & status (agen-azhan.md 3.6, screen B1/B2a).

export async function listPengajuanAgen() {
  const { data } = await client.get('/api/admin/agen/pengajuan');
  return data || [];
}

export async function setujuiAgen(jamaahId) {
  const { data } = await client.post(`/api/admin/agen/${jamaahId}/setujui`);
  return data;
}

export async function tolakAgen(jamaahId, alasan) {
  const { data } = await client.post(`/api/admin/agen/${jamaahId}/tolak`, { alasan });
  return data;
}

export async function ubahStatusAgen(jamaahId, status) {
  const { data } = await client.put(`/api/admin/agen/${jamaahId}/status`, { status });
  return data;
}

export async function verifikasiPembayaranAgen(pembayaranId) {
  const { data } = await client.post(`/api/admin/agen/pembayaran/${pembayaranId}/verifikasi`);
  return data;
}

export async function tolakPembayaranAgen(pembayaranId, alasan) {
  const { data } = await client.post(`/api/admin/agen/pembayaran/${pembayaranId}/tolak`, { alasan });
  return data;
}

export async function uploadBuktiPembayaranAgen(pembayaranId, buktiTransferUrl) {
  const { data } = await client.post(`/api/admin/agen/pembayaran/${pembayaranId}/bukti`, {
    bukti_transfer_url: buktiTransferUrl,
  });
  return data;
}

// Picker Jalur 3: agen aktif satu brand. brandId wajib untuk super admin.
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
export async function getDetailAgen(jamaahId) {
  const { data } = await client.get(`/api/admin/agen/${jamaahId}`);
  return data;
}

// B3 Riwayat Komisi brand. filter: { agen_id, jenis, dari, sampai, limit, offset }.
export async function listRiwayatKomisi(filter = {}) {
  const params = Object.fromEntries(Object.entries(filter).filter(([, v]) => v !== '' && v != null));
  const { data } = await client.get('/api/admin/agen/komisi', { params });
  return data || [];
}

// B5 Pakai Kredit Cashback di detail booking.
export async function getKreditCashback(bookingId) {
  const { data } = await client.get(`/api/admin/bookings/${bookingId}/cashback`);
  return data;
}

export async function pakaiKreditCashback(bookingId, jamaahId, nominal) {
  const { data } = await client.post(`/api/admin/bookings/${bookingId}/cashback`, { jamaah_id: jamaahId, nominal });
  return data;
}

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

// C4 Ganti Kaitan Agen (Admin Master).
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

export async function gantiKaitanAgen(jamaahId, { mode, agenJamaahId, alasan }) {
  const { data } = await client.put(`/api/admin/kaitan-agen/${jamaahId}`, {
    mode,
    agen_jamaah_id: mode === 'agen' ? agenJamaahId : null,
    alasan,
  });
  return data;
}
