import client from './client';

export async function listDestinations() {
  const response = await client.get('/api/admin/destinations');
  return response.data;
}

export async function createDestination(payload) {
  const response = await client.post('/api/admin/destinations', payload);
  return response.data;
}

export async function updateDestination(id, payload) {
  const response = await client.put(`/api/admin/destinations/${id}`, payload);
  return response.data;
}

export async function deleteDestination(id) {
  const response = await client.delete(`/api/admin/destinations/${id}`);
  return response.data;
}
