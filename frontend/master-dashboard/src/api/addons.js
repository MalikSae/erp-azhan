import client from './client';

export async function listAddOns() {
  const response = await client.get('/api/admin/addons');
  return response.data;
}

export async function createAddOn(payload) {
  const response = await client.post('/api/admin/addons', payload);
  return response.data;
}

export async function updateAddOn(id, payload) {
  const response = await client.put(`/api/admin/addons/${id}`, payload);
  return response.data;
}

export async function deleteAddOn(id) {
  const response = await client.delete(`/api/admin/addons/${id}`);
  return response.data;
}
