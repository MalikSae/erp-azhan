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
