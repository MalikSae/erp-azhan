import client from './client';

// Struktur Organisasi: pohon divisi -> tim (SPEK-PENGGUNA-ORGANISASI §5).

export async function listOrgUnits() {
  const response = await client.get('/api/admin/org-units');
  return response.data; // [{...divisi, tim: [...]}]
}

export async function createOrgUnit(payload) {
  const response = await client.post('/api/admin/org-units', payload);
  return response.data;
}

export async function updateOrgUnit(id, payload) {
  const response = await client.put(`/api/admin/org-units/${id}`, payload);
  return response.data;
}

export async function setOrgUnitStatus(id, isActive) {
  const response = await client.put(`/api/admin/org-units/${id}/status`, { is_active: isActive });
  return response.data;
}

export async function deleteOrgUnit(id) {
  const response = await client.delete(`/api/admin/org-units/${id}`);
  return response.data;
}
