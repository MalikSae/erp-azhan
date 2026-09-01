import client from './client';

export async function getStokPerlengkapan(brandId) {
  const response = await client.get('/api/admin/perlengkapan-stok', {
    params: brandId ? { brand_id: brandId } : {}
  });
  return response.data;
}

export async function markPerlengkapanDiberikan(bookingId, paxId) {
  const response = await client.put(`/api/admin/bookings/${bookingId}/pax/${paxId}/perlengkapan/distribusi`);
  return response.data;
}

export async function batalkanPerlengkapan(bookingId, paxId) {
  const response = await client.delete(`/api/admin/bookings/${bookingId}/pax/${paxId}/perlengkapan/distribusi`);
  return response.data;
}
