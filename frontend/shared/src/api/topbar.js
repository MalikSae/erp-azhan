import client from './client';

// Topbar dashboard office. Kedua endpoint cukup login admin; server menyaring per izin dan brand.

// Pencarian global: { paket: [], booking: [], jamaah: [] }, paling banyak 5 per jenis.
// Jenis yang tidak diizinkan peran dikembalikan kosong. `signal` membatalkan permintaan yang tertinggal ketikan.
export async function cariGlobal(q, { signal } = {}) {
  const { data } = await client.get('/api/admin/cari', { params: { q }, signal });
  return { paket: data?.paket || [], booking: data?.booking || [], jamaah: data?.jamaah || [] };
}

// Pekerjaan yang menunggu keputusan: { antrean: [{ jenis, jumlah }], total }.
// Hanya antrean yang boleh diproses peran yang dikembalikan.
export async function getNotifikasi() {
  const { data } = await client.get('/api/admin/notifikasi');
  return { antrean: data?.antrean || [], total: data?.total || 0 };
}
