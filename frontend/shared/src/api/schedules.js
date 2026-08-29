import api from './client';

export const listSchedulesAdmin = async (filters = {}) => {
  const params = new URLSearchParams();
  if (filters.status) params.append('status', filters.status);
  
  const { data } = await api.get(`/api/admin/schedules?${params.toString()}`);
  return data;
};

export const getScheduleAdmin = async (id) => {
  const { data } = await api.get(`/api/admin/schedules/${id}`);
  return data;
};
