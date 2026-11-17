import client from './client';

// Jelajah Haramain: tempat (global) dan ulasan jamaah.

export async function listHaramain() {
  const response = await client.get('/api/admin/haramain');
  return response.data;
}

export async function getHaramain(id) {
  const response = await client.get(`/api/admin/haramain/${id}`);
  return response.data;
}

export async function createHaramain(payload) {
  const response = await client.post('/api/admin/haramain', payload);
  return response.data;
}

export async function updateHaramain(id, payload) {
  const response = await client.put(`/api/admin/haramain/${id}`, payload);
  return response.data;
}

export async function deleteHaramain(id) {
  await client.delete(`/api/admin/haramain/${id}`);
}

export async function urutkanHaramain(ids) {
  await client.put('/api/admin/haramain/urutan', { ids });
}

export async function listUlasanHaramain(tab) {
  const response = await client.get('/api/admin/haramain-ulasan', { params: { tab } });
  return response.data;
}

export async function aturTampilUlasan(id, disembunyikan) {
  await client.put(`/api/admin/haramain-ulasan/${id}/tampil`, { disembunyikan });
}

export async function abaikanLaporanUlasan(id) {
  await client.post(`/api/admin/haramain-ulasan/${id}/abaikan`);
}
