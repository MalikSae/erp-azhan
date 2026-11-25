import client from './client';

// Pengguna (SPEK-AKSES-PENGGUNA-2026-11-25 §5). Create/update membawa profil, akses
// brand, jenis akses, dan izin sekaligus; diproses atomik di backend.

export async function listAdminUsers() {
  const response = await client.get('/api/admin/users');
  return response.data;
}

export async function getAdminUser(id) {
  const response = await client.get(`/api/admin/users/${id}`);
  return response.data;
}

export async function createAdminUser(payload) {
  const response = await client.post('/api/admin/users', payload);
  return response.data;
}

export async function updateAdminUser(id, payload) {
  const response = await client.put(`/api/admin/users/${id}`, payload);
  return response.data;
}

export async function setAdminUserStatus(id, isActive) {
  const response = await client.put(`/api/admin/users/${id}/status`, { is_active: isActive });
  return response.data;
}

export async function resetAdminUserPassword(id, password) {
  const response = await client.put(`/api/admin/users/${id}/password`, { password });
  return response.data;
}

export async function deleteAdminUser(id) {
  const response = await client.delete(`/api/admin/users/${id}`);
  return response.data;
}
