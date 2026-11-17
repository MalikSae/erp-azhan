import client from './client';

// Manasik: satu manasik untuk satu atau lebih paket, rekap kehadiran, dan absensi hari-H.

export async function listManasik(params = {}) {
  const response = await client.get('/api/admin/manasik', { params });
  return response.data;
}

export async function listPaketTersediaManasik(params = {}) {
  const response = await client.get('/api/admin/manasik/paket-tersedia', { params });
  return response.data;
}

export async function getManasik(id) {
  const response = await client.get(`/api/admin/manasik/${id}`);
  return response.data;
}

export async function createManasik(payload) {
  const response = await client.post('/api/admin/manasik', payload);
  return response.data;
}

export async function updateManasik(id, payload) {
  const response = await client.put(`/api/admin/manasik/${id}`, payload);
  return response.data;
}

export async function deleteManasik(id) {
  await client.delete(`/api/admin/manasik/${id}`);
}

export async function tambahPaketManasik(id, scheduleIds) {
  const response = await client.post(`/api/admin/manasik/${id}/paket`, { schedule_ids: scheduleIds });
  return response.data;
}

export async function lepasPaketManasik(id, scheduleId) {
  const response = await client.delete(`/api/admin/manasik/${id}/paket/${scheduleId}`);
  return response.data;
}

export async function catatHadirManasik(id, paxId, hadir) {
  const response = await client.put(`/api/admin/manasik/${id}/absensi`, { pax_id: paxId, hadir });
  return response.data;
}
