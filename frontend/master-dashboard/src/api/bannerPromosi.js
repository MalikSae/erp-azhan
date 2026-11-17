import client from './client';

// Banner Promosi Beranda app Hana.

export async function listBannerPromosi(brandId) {
  const response = await client.get('/api/admin/banner-promosi', { params: brandId ? { brand_id: brandId } : {} });
  return response.data;
}

export async function createBannerPromosi(payload) {
  const response = await client.post('/api/admin/banner-promosi', payload);
  return response.data;
}

export async function updateBannerPromosi(id, payload) {
  const response = await client.put(`/api/admin/banner-promosi/${id}`, payload);
  return response.data;
}

export async function deleteBannerPromosi(id) {
  await client.delete(`/api/admin/banner-promosi/${id}`);
}

export async function urutkanBannerPromosi(brandId, ids) {
  await client.put('/api/admin/banner-promosi/urutan', { brand_id: brandId || undefined, ids });
}

