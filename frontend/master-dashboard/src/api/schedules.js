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

export async function updateScheduleSeat(id, seatSisa) {
  const response = await client.put(`/api/admin/schedules/${id}/seat`, { seat_sisa: seatSisa });
  return response.data;
}

export async function deleteSchedule(id) {
  const response = await client.delete(`/api/admin/schedules/${id}`);
  return response.data;
}
