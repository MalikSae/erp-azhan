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
