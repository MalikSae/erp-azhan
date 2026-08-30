import client from './client';

export async function listAirports(search = '') {
  const params = search ? { search } : {};
  const response = await client.get('/api/admin/airports', { params });
  return response.data;
}

export async function createAirport(payload) {
  const response = await client.post('/api/admin/airports', payload);
  return response.data;
}

export async function updateAirport(id, payload) {
  const response = await client.put(`/api/admin/airports/${id}`, payload);
  return response.data;
}

export async function deleteAirport(id) {
  const response = await client.delete(`/api/admin/airports/${id}`);
  return response.data;
}
