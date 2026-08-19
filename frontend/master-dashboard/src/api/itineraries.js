import client from './client';

export async function listItineraries() {
  const response = await client.get('/api/admin/itineraries');
  return response.data;
}

export async function getItinerary(id) {
  const response = await client.get(`/api/admin/itineraries/${id}`);
  return response.data;
}

export async function createItinerary(payload) {
  const response = await client.post('/api/admin/itineraries', payload);
  return response.data;
}

export async function updateItinerary(id, payload) {
  const response = await client.put(`/api/admin/itineraries/${id}`, payload);
  return response.data;
}

export async function deleteItinerary(id) {
  const response = await client.delete(`/api/admin/itineraries/${id}`);
  return response.data;
}
