import client from './client';

export async function getStokPerlengkapan(brandId) {
  const response = await client.get('/api/admin/perlengkapan-stok', {
    params: brandId ? { brand_id: brandId } : {}
  });
  return response.data;
}

export async function markPerlengkapanDiberikan(bookingId) {
  const response = await client.put(`/api/admin/bookings/${bookingId}/perlengkapan/distribusi`);
  return response.data;
}

export async function batalkanPerlengkapan(bookingId) {
  const response = await client.delete(`/api/admin/bookings/${bookingId}/perlengkapan/distribusi`);
  return response.data;
}
