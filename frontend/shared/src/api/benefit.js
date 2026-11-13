import client from './client';

// Highlight benefit paket: master per brand (ikon + teks, maksimal 30 karakter). Tiap paket memilih tepat empat.
// Izin master: masterdata.view / masterdata.edit. Form paket memakai endpoint "benefit-options" (schedule.view /
// schedule.edit) supaya admin paket bisa memilih dan menambah benefit tanpa izin master.

// brandId wajib untuk admin holding; admin satu brand selalu menerima brand-nya sendiri.
export async function listBenefits(brandId) {
  const { data } = await client.get('/api/admin/benefits', { params: brandId ? { brand_id: brandId } : {} });
  return data || [];
}

// Daftar ikon tetap yang boleh dipilih: [{ key, label }].
export async function listBenefitIcons() {
  const { data } = await client.get('/api/admin/benefits/ikon');
  return data || [];
}

// payload: { brand_id (admin holding), teks, ikon }
export async function createBenefit(payload) {
  const { data } = await client.post('/api/admin/benefits', payload);
  return data;
}

// payload: { teks, ikon, is_active }. Perubahan berlaku di semua paket yang memakai benefit ini.
export async function updateBenefit(id, payload) {
  const { data } = await client.put(`/api/admin/benefits/${id}`, payload);
  return data;
}

// Ditolak server (409) bila benefit masih dipakai paket; nonaktifkan saja.
export async function deleteBenefit(id) {
  const { data } = await client.delete(`/api/admin/benefits/${id}`);
  return data;
}

// Untuk form paket: { ikon: [{ key, label }], benefits: [benefit aktif brand] }.
export async function getBenefitOptions(brandId) {
  const { data } = await client.get('/api/admin/schedules/benefit-options', { params: brandId ? { brand_id: brandId } : {} });
  return { ikon: data?.ikon || [], benefits: data?.benefits || [] };
}

// Menambah benefit baru dari form paket. payload: { brand_id (admin holding), teks, ikon }
export async function createBenefitOption(payload) {
  const { data } = await client.post('/api/admin/schedules/benefit-options', payload);
  return data;
}
