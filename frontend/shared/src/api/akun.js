import client from './client';

// Akun Pelanggan (layar admin, hanya baca): akun yang bisa booking di situs publik.

export async function listAkunPelanggan({ q, brandId, statusAgen, upline, page, perPage } = {}) {
  const params = { page, per_page: perPage };
  if (q) params.q = q;
  if (brandId) params.brand_id = brandId;
  if (statusAgen) params.status_agen = statusAgen;
  if (upline) params.upline = upline;
  const { data } = await client.get('/api/admin/akun', { params });
  return data;
}

export async function getAkunPelanggan(akunId) {
  const { data } = await client.get(`/api/admin/akun/${akunId}`);
  return data;
}

// Admin mengubah data akun. Alasan wajib; API membalas 409 dengan akun_id pemakai bila nilai baru bentrok.
export async function ubahProfilAkun(akunId, { nama_lengkap, no_hp, domisili, alasan }) {
  const { data } = await client.put(`/api/admin/akun/${akunId}`, { nama_lengkap, no_hp, domisili, alasan });
  return data;
}

export async function ubahNikAgen(akunId, { nik, alasan }) {
  const { data } = await client.put(`/api/admin/akun/${akunId}/nik-agen`, { nik, alasan });
  return data;
}
