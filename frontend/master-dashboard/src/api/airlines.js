import client from './client';

export async function listAirlines() {
  const response = await client.get('/api/admin/airlines');
  return response.data;
}

export async function createAirline(payload) {
  const response = await client.post('/api/admin/airlines', payload);
  return response.data;
}

export async function updateAirline(id, payload) {
  const response = await client.put(`/api/admin/airlines/${id}`, payload);
  return response.data;
}

export async function deleteAirline(id) {
  const response = await client.delete(`/api/admin/airlines/${id}`);
  return response.data;
}
