import client from './client';

export async function listHotels() {
  const response = await client.get('/api/admin/hotels');
  return response.data;
}

export async function createHotel(payload) {
  const response = await client.post('/api/admin/hotels', payload);
  return response.data;
}

export async function updateHotel(id, payload) {
  const response = await client.put(`/api/admin/hotels/${id}`, payload);
  return response.data;
}

export async function deleteHotel(id) {
  const response = await client.delete(`/api/admin/hotels/${id}`);
  return response.data;
}
