import client from './client';

export async function listBrands() {
  const response = await client.get('/api/admin/brands');
  return response.data;
}

export async function getBrand(id) {
  const response = await client.get(`/api/admin/brands/${id}`);
  return response.data;
}

export async function createBrand(payload) {
  const response = await client.post('/api/admin/brands', payload);
  return response.data;
}

export async function updateBrand(id, payload) {
  const response = await client.put(`/api/admin/brands/${id}`, payload);
  return response.data;
}

export async function deleteBrand(id) {
  const response = await client.delete(`/api/admin/brands/${id}`);
  return response.data;
}

export async function getMyBrand() {
  const response = await client.get('/api/admin/my-brand');
  return response.data;
}

