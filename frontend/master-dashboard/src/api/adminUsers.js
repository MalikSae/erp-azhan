import client from './client';

export async function listAdminUsers() {
  const response = await client.get('/api/admin/users');
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

export async function resetAdminUserPassword(id, password) {
  const response = await client.put(`/api/admin/users/${id}/password`, { password });
  return response.data;
}

export async function deleteAdminUser(id) {
  const response = await client.delete(`/api/admin/users/${id}`);
  return response.data;
}
