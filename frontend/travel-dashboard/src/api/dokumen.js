import client from './client';

export async function listDokumen(jamaahId) {
  const response = await client.get(`/api/admin/jamaah/${jamaahId}/dokumen`);
  return response.data;
}

export async function upsertDokumen(jamaahId, { jenis, file_url }) {
  const response = await client.post(`/api/admin/jamaah/${jamaahId}/dokumen`, { jenis, file_url });
  return response.data;
}

export async function updateDokumenStatus(id, status) {
  const response = await client.put(`/api/admin/dokumen/${id}/status`, { status });
  return response.data;
}
