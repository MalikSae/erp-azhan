import client from './client';

// RBAC Fase 1 — endpoint di balik permission `user.admin`.

export async function listRoles() {
  const response = await client.get('/api/admin/roles');
  return response.data;
}

export async function listPermissions() {
  const response = await client.get('/api/admin/permissions');
  return response.data;
}

export async function getUserRoles(userId) {
  const response = await client.get(`/api/admin/users/${userId}/roles`);
  return response.data; // { roles: [...] }
}

export async function setUserRoles(userId, roles) {
  const response = await client.put(`/api/admin/users/${userId}/roles`, { roles });
  return response.data; // { roles: [...] }
}
