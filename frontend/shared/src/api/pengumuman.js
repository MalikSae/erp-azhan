import client from './client';

// Pengumuman untuk pelanggan (tampil di dashboard akun web). Izin: pengumuman.manage.

export async function listPengumuman(brandId) {
  const { data } = await client.get('/api/admin/pengumuman', { params: brandId ? { brand_id: brandId } : {} });
  return data || [];
}

// payload: { brand_id (admin holding), schedule_id (opsional), judul, isi, mulai, akhir } dengan tanggal YYYY-MM-DD.
export async function createPengumuman(payload) {
  const { data } = await client.post('/api/admin/pengumuman', payload);
  return data;
}

export async function updatePengumuman(id, payload) {
  const { data } = await client.put(`/api/admin/pengumuman/${id}`, payload);
  return data;
}

export async function deletePengumuman(id) {
  const { data } = await client.delete(`/api/admin/pengumuman/${id}`);
  return data;
}
