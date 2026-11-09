import client from './client';

// Persetujuan perubahan data jamaah dari pelanggan (halaman Data Jamaah di akun). Izin: jamaah.perubahan.approve.

// status: 'menunggu' (bawaan) atau 'semua' untuk riwayat; jamaahId menyaring satu jamaah.
export async function listPerubahanJamaah({ status = 'menunggu', jamaahId } = {}) {
  const params = { status };
  if (jamaahId) params.jamaah_id = jamaahId;
  const { data } = await client.get('/api/admin/perubahan-jamaah', { params });
  return data || [];
}

export async function getPerubahanJamaah(id) {
  const { data } = await client.get(`/api/admin/perubahan-jamaah/${id}`);
  return data;
}

export async function setujuiPerubahanJamaah(id) {
  const { data } = await client.post(`/api/admin/perubahan-jamaah/${id}/setujui`);
  return data;
}

export async function tolakPerubahanJamaah(id, catatan) {
  const { data } = await client.post(`/api/admin/perubahan-jamaah/${id}/tolak`, { catatan });
  return data;
}
