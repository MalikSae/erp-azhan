import client from './client';

export async function listCategories(params = {}) {
  const response = await client.get('/api/admin/categories', { params });
  return response.data;
}

export async function getCategory(id) {
  const response = await client.get(`/api/admin/categories/${id}`);
  return response.data;
}

export async function createCategory(payload) {
  const response = await client.post('/api/admin/categories', payload);
  return response.data;
}

export async function updateCategory(id, payload) {
  const response = await client.put(`/api/admin/categories/${id}`, payload);
  return response.data;
}

export async function deleteCategory(id) {
  const response = await client.delete(`/api/admin/categories/${id}`);
  return response.data;
}
