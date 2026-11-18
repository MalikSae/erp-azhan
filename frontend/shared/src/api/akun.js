import client from './client';

// Akun (layar admin, hanya baca): akun yang bisa booking di situs publik.

// agen: 'agen' (aktif atau nonaktif) | 'bukan'.
export async function listAkunPelanggan({ q, brandId, statusAgen, upline, agen, page, perPage } = {}) {
  const params = { page, per_page: perPage };
  if (q) params.q = q;
  if (brandId) params.brand_id = brandId;
  if (statusAgen) params.status_agen = statusAgen;
  if (upline) params.upline = upline;
  if (agen) params.agen = agen;
  const { data } = await client.get('/api/admin/akun', { params });
  return data;
}

export async function getAkunPelanggan(akunId) {
  const { data } = await client.get(`/api/admin/akun/${akunId}`);
  return data;
}

// Admin mengubah data akun. Alasan wajib; API membalas 409 dengan akun_id pemakai bila nilai baru bentrok.
export async function ubahProfilAkun(akunId, { nama_lengkap, no_hp, domisili, alasan }) {
  const { data } = await client.put(`/api/admin/akun/${akunId}`, { nama_lengkap, no_hp, domisili, alasan });
  return data;
}

export async function ubahNikAgen(akunId, { nik, alasan }) {
  const { data } = await client.put(`/api/admin/akun/${akunId}/nik-agen`, { nik, alasan });
  return data;
}

// Aktivasi akun jamaah (admin). Alasan wajib; galat API membawa pesan siap tampil di response.data.error.
export async function lepasTautanAkun(akunId, { alasan }) {
  const { data } = await client.post(`/api/admin/akun/${akunId}/lepas-tautan`, { alasan });
  return data;
}

export async function kosongkanNikAgen(akunId, { alasan }) {
  const { data } = await client.post(`/api/admin/akun/${akunId}/kosongkan-nik-agen`, { alasan });
  return data;
}

// Akun agen (pengajuan, aktif, nonaktif) yang tertaut ke jamaah, untuk audit.
export async function listAgenDariTautan({ brandId, page, perPage } = {}) {
  const params = { page, per_page: perPage };
  if (brandId) params.brand_id = brandId;
  const { data } = await client.get('/api/admin/akun/agen-dari-tautan', { params });
  return data;
}

// Buka kunci percobaan aktivasi satu kode booking atau satu NIK (salah satu). Admin Master wajib mengirim brandId.
export async function resetHitungAktivasi({ kodeBooking, nik, brandId }) {
  const body = kodeBooking ? { kode_booking: kodeBooking } : { nik };
  const { data } = await client.post('/api/admin/akun/aktivasi/reset-hitung', body, { params: brandId ? { brand_id: brandId } : undefined });
  return data;
}
