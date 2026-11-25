import client from './client';

export async function listSchedulesAdmin() {
  const response = await client.get('/api/admin/schedules');
  return response.data;
}

export async function getSchedule(id) {
  const response = await client.get(`/api/admin/schedules/${id}`);
  return response.data;
}

export async function createSchedule(payload) {
  const response = await client.post('/api/admin/schedules', payload);
  return response.data;
}

export async function updateSchedule(id, payload) {
  const response = await client.put(`/api/admin/schedules/${id}`, payload);
  return response.data;
}

export async function updateScheduleStatus(id, status) {
  const response = await client.put(`/api/admin/schedules/${id}/status`, { status });
  return response.data;
}

export async function getScheduleSeat(id) {
  const response = await client.get(`/api/admin/schedules/${id}/seat`);
  return response.data;
}

// Mengisi sisa seat = kuota − seat dipegang booking (tanpa angka bebas).
export async function samakanScheduleSeat(id) {
  const response = await client.post(`/api/admin/schedules/${id}/seat/samakan`);
  return response.data;
}

// seatTampil null mematikan angka tampil di publik.
export async function updateScheduleSeatTampil(id, seatTampil) {
  const response = await client.put(`/api/admin/schedules/${id}/seat-tampil`, { seat_tampil: seatTampil });
  return response.data;
}

export async function deleteSchedule(id) {
  const response = await client.delete(`/api/admin/schedules/${id}`);
  return response.data;
}

// Ringkasan manasik paket ({ manasik: null } bila belum ada). Manasik dikelola di menu Manasik (api/manasik.js).
export async function getScheduleManasik(id) {
  const response = await client.get(`/api/admin/schedules/${id}/manasik`);
  return response.data;
}

// Pilihan "Lokasi Dikunjungi": tempat Jelajah Haramain yang aktif [{ id, nama, kota, foto_url }].
export async function getScheduleLokasiOptions() {
  const response = await client.get('/api/admin/schedules/lokasi-options');
  return response.data;
}
