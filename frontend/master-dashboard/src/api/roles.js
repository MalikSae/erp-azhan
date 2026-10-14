import client from './client';

// Daftar role RBAC beserta scope & permission-nya (untuk form Pengguna).
// Penggantian role pengguna dilakukan lewat form Pengguna (atomik).
export async function listRoles() {
  const response = await client.get('/api/admin/roles');
  return response.data;
}
