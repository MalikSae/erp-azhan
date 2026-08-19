import client from './client';

// ─── Item Katalog Global ──────────────────────────────────────────────────────

export async function listPerlengkapanItems() {
  const response = await client.get('/api/admin/perlengkapan-items');
  return response.data;
}

export async function createPerlengkapanItem(payload) {
  const response = await client.post('/api/admin/perlengkapan-items', payload);
  return response.data;
}

export async function updatePerlengkapanItem(id, payload) {
  const response = await client.put(`/api/admin/perlengkapan-items/${id}`, payload);
  return response.data;
}

export async function deletePerlengkapanItem(id) {
  const response = await client.delete(`/api/admin/perlengkapan-items/${id}`);
  return response.data;
}

// ─── Stok Per Brand ───────────────────────────────────────────────────────────

export async function listPerlengkapanStok(brandId) {
  const params = brandId ? { brand_id: brandId } : {};
  const response = await client.get('/api/admin/perlengkapan-stok', { params });
  return response.data;
}

export async function updatePerlengkapanStok(itemId, brandId, payload) {
  const response = await client.put(`/api/admin/perlengkapan-stok/${itemId}`, payload, {
    params: { brand_id: brandId }
  });
  return response.data;
}

// ─── Set Template Global ─────────────────────────────────────────────────────

export async function getPerlengkapanSetTemplate() {
  const response = await client.get('/api/admin/perlengkapan-set-template');
  return response.data;
}

export async function updatePerlengkapanSetTemplate(items) {
  const response = await client.put('/api/admin/perlengkapan-set-template', items);
  return response.data;
}
